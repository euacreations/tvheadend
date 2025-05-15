package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/euacreations/tvheadend/internal/database"
	"github.com/euacreations/tvheadend/internal/models"
	"github.com/euacreations/tvheadend/pkg/ffmpeg"
)

type PlaylistExecutor struct {
	repo       *database.Repository
	ffmpeg     *ffmpeg.Streamer
	mediaCache map[sql.NullInt64]*models.MediaFile
	cacheMux   sync.RWMutex
	//positionUpdateMux sync.Mutex
	currentState struct {
		playlist      *models.Playlist
		items         []*models.PlaylistItem
		currentIndex  int
		startOffset   int
		nextIndex     int
		playlistStart time.Time
		streamCancel  context.CancelFunc
		streamMux     sync.Mutex
	}
}

func NewPlaylistExecutor(repo *database.Repository, ffmpeg *ffmpeg.Streamer) *PlaylistExecutor {
	return &PlaylistExecutor{
		repo:       repo,
		ffmpeg:     ffmpeg,
		mediaCache: make(map[sql.NullInt64]*models.MediaFile),
	}
}
func (e *PlaylistExecutor) Execute(ctx context.Context, channel *models.Channel) error {
	if err := e.initializePlaylist(ctx, channel); err != nil {
		return fmt.Errorf("playlist initialization failed: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			e.cleanup()
			return nil
		default:
			// Calculate time until next day's playlist starts
			nextDayStart := calculateNextDayStart(time.Now(), channel.StartTime)
			timeUntilTransition := time.Until(nextDayStart)
			currentItem := e.currentState.items[e.currentState.currentIndex]

			// Get duration based on item type
			var maxDuration int
			var inputPath string
			//var startOffset int
			var err error

			switch currentItem.Type {
			case models.PlaylistItemTypeMedia:
				media, err := e.getMediaFile(ctx, currentItem.MediaID)
				if err != nil {
					return fmt.Errorf("media lookup failed: %w", err)
				}
				inputPath = filepath.Join(channel.StorageRoot, "media", media.FilePath)
				maxDuration = media.DurationSeconds
				//startOffset = e.currentState.startOffset

			case models.PlaylistItemTypeUDP:
				if !currentItem.StreamID.Valid {
					return fmt.Errorf("invalid stream ID for UDP item")
				}
				stream, err := e.repo.GetUDPStream(ctx, currentItem.StreamID)
				if err != nil {
					return fmt.Errorf("stream lookup failed: %w", err)
				}
				inputPath = stream.StreamURL

				// Calculate effective duration
				originalOffset := e.currentState.startOffset
				if stream.DurationSeconds != nil {
					// Finite stream: remaining duration = total duration - offset
					maxDuration = *stream.DurationSeconds - originalOffset
					if maxDuration < 0 {
						maxDuration = 0
					}
				} else {
					// Infinite stream: duration = time until transition - offset
					maxDuration = int(timeUntilTransition.Seconds()) - originalOffset
					if maxDuration < 0 {
						maxDuration = 0
					}
				}
				e.currentState.startOffset = 0
				//startOffset = 0

				// Final safety check
				if maxDuration < 0 {
					maxDuration = 0
				}

			}

			// Cap duration at time until transition
			if timeUntilTransition < time.Duration(maxDuration)*time.Second {
				maxDuration = int(timeUntilTransition.Seconds())
			}

			// Queue next item while playing current
			prepDone := make(chan struct{})
			go func() {
				defer close(prepDone)
				e.unlockItem(currentItem)

				// Re-fetch playlist if needed
				items, err := e.repo.GetPlaylistItems(ctx, e.currentState.playlist.PlaylistID)
				if err == nil && len(items) > 0 {
					e.currentState.items = items
				}

				// Prepare next item
				nextIndex := (e.currentState.currentIndex + 1) % len(e.currentState.items)
				e.currentState.nextIndex = nextIndex
				e.lockItem(e.currentState.items[nextIndex])
			}()

			// Play current item
			err = e.playItem(ctx, channel, currentItem, inputPath, e.currentState.startOffset, maxDuration)
			e.currentState.startOffset = 0

			<-prepDone // Ensure next item was prepared

			if err != nil {
				return fmt.Errorf("playback failed: %w", err)
			}

			// Check if it's time to transition
			if time.Now().After(nextDayStart) {
				if err := e.transitionToNextPlaylist(ctx, channel); err != nil {
					return fmt.Errorf("playlist transition failed: %w", err)
				}
				continue
			}

			// Move to next item
			e.currentState.currentIndex = e.currentState.nextIndex
		}
	}
}

func (e *PlaylistExecutor) initializePlaylist(ctx context.Context, channel *models.Channel) error {
	effectiveDate := calculateEffectiveDate(time.Now(), channel.StartTime)

	playlist, err := e.repo.GetPlaylistForDate(ctx, channel.ChannelID, effectiveDate)

	maxFallbackDays := 7
	daysTried := 0
	if val, ok := os.LookupEnv("MAX_PLAYLIST_FALLBACK_DAYS"); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxFallbackDays = parsed
		}
	}

	for err != nil {
		if !channel.UsePreviousDayFallback || daysTried >= maxFallbackDays {
			return fmt.Errorf("no playlist found after fallback attempts: %w", err)
		}

		effectiveDate = effectiveDate.Add(-24 * time.Hour)
		daysTried++
		playlist, err = e.repo.GetPlaylistForDate(ctx, channel.ChannelID, effectiveDate)
	}

	items, err := e.repo.GetPlaylistItems(ctx, playlist.PlaylistID)
	if err != nil {
		return fmt.Errorf("failed to get items: %w", err)
	}

	if len(items) == 0 {
		return fmt.Errorf("empty playlist")
	}

	fmt.Println("Effective Date:", effectiveDate)
	fmt.Println("Channel Start Time:", channel.StartTime)

	startIndex, startOffset := e.calculateStartPosition(ctx, items, effectiveDate)
	//fmt.Printf("Starting playback from item %d with offset %d seconds\n", startIndex, startOffset)
	e.currentState.playlist = playlist
	e.currentState.items = items
	e.currentState.currentIndex = startIndex
	e.currentState.playlistStart = effectiveDate
	e.currentState.startOffset = startOffset

	// Lock initial items
	e.lockItem(items[startIndex])
	e.currentState.nextIndex = (startIndex + 1) % len(items)
	e.lockItem(items[e.currentState.nextIndex])

	return nil
}

func (e *PlaylistExecutor) playItem(ctx context.Context, channel *models.Channel,
	item *models.PlaylistItem, inputPath string, offset int, maxDuration int) error {

	e.currentState.streamMux.Lock()
	defer e.currentState.streamMux.Unlock()

	// Reset the streamer before starting new stream
	e.ffmpeg.Reset()
	e.ffmpeg.SetProgressCallback(nil)

	// Build FFmpeg config
	config := ffmpeg.StreamConfig{
		InputPath:               inputPath,
		InputType:               item.Type,
		OutputURL:               channel.OutputUDP,
		StartOffset:             time.Duration(offset) * time.Second,
		Duration:                time.Duration(maxDuration) * time.Second,
		VideoCodec:              channel.VideoCodec,
		VideoBitrate:            channel.VideoBitrate,
		MinBitrate:              channel.MinBitrate,
		MaxBitrate:              channel.MaxBitrate,
		AudioCodec:              channel.AudioCodec,
		AudioBitrate:            channel.AudioBitrate,
		BufferSize:              channel.BufferSize,
		OutputResolution:        channel.OutputResolution,
		PacketSize:              channel.PacketSize,
		MpegTSOriginalNetworkID: channel.MPEGTSOriginalNetworkID,
		MpegTSTransportStreamID: channel.MPEGTSTransportStreamID,
		MpegTSServiceID:         channel.MPEGTSServiceID,
		MpegTSStartPID:          channel.MPEGTSStartPID,
		MpegTSPMTStartPID:       channel.MPEGTSPMTStartPID,
		MetadataServiceProvider: channel.MetadataServiceProvider,
		MmetadataServiceName:    channel.ChannelName,
	}

	// Add overlays
	overlays, _ := e.repo.GetChannelOverlays(ctx, channel.ChannelID)
	for _, overlay := range overlays {
		config.Overlays = append(config.Overlays, models.Overlay{
			Type:      overlay.Type,
			FilePath:  filepath.Join(channel.StorageRoot, "data", overlay.FilePath),
			Text:      overlay.Text,
			PositionX: overlay.PositionX,
			PositionY: overlay.PositionY,
			FontSize:  overlay.FontSize,
			FontColor: overlay.FontColor,
		})
	}

	// Add program name overlay if this is a media file
	if item.Type == models.PlaylistItemTypeMedia {
		media, err := e.getMediaFile(ctx, item.MediaID)
		if err == nil && media.ProgramName != "" {
			programNameOverlay := models.Overlay{
				Type:      "text",
				Text:      media.ProgramName,
				PositionX: "W/12",
				PositionY: "H/12",
				FontSize:  "H/45",
				FontColor: "white",
				FontFile:  filepath.Join(channel.StorageRoot, "data", "NotoSansSinhala-Regular.ttf"),
			}
			config.Overlays = append(config.Overlays, programNameOverlay)
		}
	}

	// Create cancelable context
	streamCtx, cancel := context.WithCancel(ctx)
	e.currentState.streamCancel = cancel

	e.ffmpeg.SetProgressCallback(func(position float64) {
		state := &models.ChannelState{
			ChannelID:         channel.ChannelID,
			CurrentPlaylistID: item.PlaylistID,
			CurrentItemID:     item.ItemID,
			CurrentPosition:   position,
			LastUpdateTime:    time.Now(),
			Running:           true,
			FFmpegPID:         e.ffmpeg.PID(),
		}

		if err := e.repo.UpdateChannelState(context.Background(), state); err != nil {
			log.Printf("Failed to update position: %v", err)
		}
	})

	// Start FFmpeg stream
	if err := e.ffmpeg.Start(streamCtx, config); err != nil {
		return err
	}

	// Update channel state
	state := &models.ChannelState{
		ChannelID:         channel.ChannelID,
		CurrentPlaylistID: item.PlaylistID,
		CurrentItemID:     item.ItemID,
		CurrentPosition:   float64(offset),
		Running:           true,
		FFmpegPID:         e.ffmpeg.PID(),
		LastUpdateTime:    time.Now(),
	}
	if err := e.repo.UpdateChannelState(ctx, state); err != nil {
		e.ffmpeg.Stop()
		return fmt.Errorf("state update failed: %w", err)
	}

	// Wait for completion or context cancellation
	select {
	case <-streamCtx.Done():
		return streamCtx.Err()
	case <-e.ffmpeg.Done():
		return nil
	}
}

func (e *PlaylistExecutor) transitionToNextPlaylist(ctx context.Context, channel *models.Channel) error {
	// Stop current stream
	e.currentState.streamMux.Lock()
	if e.currentState.streamCancel != nil {
		e.currentState.streamCancel()
	}
	e.currentState.streamMux.Unlock()

	// Get next day's playlist
	nextDay := calculateNextDayStart(time.Now(), channel.StartTime)
	playlist, err := e.repo.GetPlaylistForDate(ctx, channel.ChannelID, nextDay)
	if err != nil && !channel.UsePreviousDayFallback {
		return fmt.Errorf("next playlist unavailable: %w", err)
	}

	// Fallback to current playlist if needed
	if playlist == nil {
		playlist = e.currentState.playlist
	}

	// Reinitialize with new playlist
	e.currentState.playlist = playlist
	e.currentState.currentIndex = 0
	e.currentState.playlistStart = nextDay

	// Get fresh items list
	items, err := e.repo.GetPlaylistItems(ctx, playlist.PlaylistID)
	if err != nil {
		return fmt.Errorf("failed to refresh items: %w", err)
	}
	e.currentState.items = items

	// Lock new items
	e.lockItem(items[0])
	e.currentState.nextIndex = 1 % len(items)
	e.lockItem(items[e.currentState.nextIndex])

	return nil
}

func (e *PlaylistExecutor) lockItem(item *models.PlaylistItem) {
	_ = e.repo.LockPlaylistItem(context.Background(), item.ItemID)
}

func (e *PlaylistExecutor) unlockItem(item *models.PlaylistItem) {
	_ = e.repo.UnlockPlaylistItem(context.Background(), item.ItemID)
}

func (e *PlaylistExecutor) getMediaFile(ctx context.Context, mediaID sql.NullInt64) (*models.MediaFile, error) {
	e.cacheMux.RLock()
	if media, exists := e.mediaCache[mediaID]; exists {
		e.cacheMux.RUnlock()
		return media, nil
	}
	e.cacheMux.RUnlock()

	media, err := e.repo.GetMediaFile(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	e.cacheMux.Lock()
	e.mediaCache[mediaID] = media
	e.cacheMux.Unlock()

	return media, nil
}

func (e *PlaylistExecutor) cleanup() {
	e.currentState.streamMux.Lock()
	defer e.currentState.streamMux.Unlock()

	if e.currentState.streamCancel != nil {
		e.currentState.streamCancel()
	}

	// Unlock all items
	for _, item := range e.currentState.items {
		e.unlockItem(item)
	}
}

// Helper functions
func calculateEffectiveDate(now time.Time, startTime time.Time) time.Time {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, now.Location())

	if now.Before(todayStart) {
		return todayStart.Add(-24 * time.Hour)
	}
	return todayStart
}

func calculateNextDayStart(now time.Time, startTime time.Time) time.Time {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, now.Location())

	if now.After(todayStart) {
		return todayStart.Add(24 * time.Hour)
	}
	return todayStart
}

func (e *PlaylistExecutor) calculateStartPosition(ctx context.Context, items []*models.PlaylistItem,
	playlistStart time.Time) (int, int) {

	elapsed := time.Since(playlistStart)
	if elapsed < 0 {
		return 0, 0
	}

	totalDuration := 0
	for _, item := range items {
		var duration int

		switch item.Type {
		case models.PlaylistItemTypeMedia:
			media, err := e.getMediaFile(ctx, item.MediaID)
			if err != nil {
				return 0, 0
			}
			duration = media.DurationSeconds

		case models.PlaylistItemTypeUDP:
			// Get UDP stream duration (NULL means infinite)
			if item.StreamID.Valid {
				stream, err := e.repo.GetUDPStream(ctx, item.StreamID)
				if err != nil {
					return 0, 0
				}
				if stream.DurationSeconds != nil {
					duration = *stream.DurationSeconds
				} else {
					// Infinite stream - use remaining time in playlist
					duration = 24 * 3600 // Max 24 hours
				}
			} else {
				duration = 24 * 3600 // Default to 24 hours if stream ID is invalid
			}
		}

		totalDuration += duration
	}

	positionSec := int(elapsed.Seconds()) % (24 * 3600)
	if totalDuration > 0 {
		positionSec %= totalDuration
	}

	accumulated := 0
	for i, item := range items {
		var duration int

		switch item.Type {
		case models.PlaylistItemTypeMedia:
			media, err := e.getMediaFile(ctx, item.MediaID)
			if err != nil {
				return 0, 0
			}
			duration = media.DurationSeconds

		case models.PlaylistItemTypeUDP:
			if item.StreamID.Valid {
				stream, err := e.repo.GetUDPStream(ctx, item.StreamID)
				if err != nil {
					return 0, 0
				}
				if stream.DurationSeconds != nil {
					duration = *stream.DurationSeconds
				} else {
					duration = 24 * 3600
				}
			} else {
				duration = 24 * 3600
			}
		}

		if accumulated+duration > positionSec {
			return i, positionSec - accumulated
		}
		accumulated += duration
	}

	return 0, 0
}

// func (e *PlaylistExecutor) calculateStartPosition(ctx context.Context, items []*models.PlaylistItem,
// 	playlistStart time.Time) (int, int) {

// 	elapsed := time.Since(playlistStart)
// 	if elapsed < 0 {
// 		return 0, 0
// 	}

// 	totalDuration := 0
// 	for _, item := range items {
// 		media, err := e.getMediaFile(ctx, item.MediaID)
// 		if err != nil {
// 			return 0, 0
// 		}

// 		totalDuration += media.DurationSeconds
// 	}

// 	positionSec := int(elapsed.Seconds()) % (24 * 3600)
// 	if totalDuration > 0 {
// 		positionSec %= totalDuration
// 	}

// 	accumulated := 0
// 	for i, item := range items {
// 		media, err := e.getMediaFile(ctx, item.MediaID)
// 		if err != nil {
// 			return 0, 0
// 		}

// 		if accumulated+media.DurationSeconds > positionSec {
// 			return i, positionSec - accumulated
// 		}
// 		accumulated += media.DurationSeconds
// 	}

// 	return 0, 0
// }

func (e *PlaylistExecutor) GetPlaylists(ctx context.Context, channelID int) ([]*models.Playlist, error) {
	playlist, err := e.repo.GetPlaylists(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlists: %w", err)
	}

	return playlist, err
}

func (e *PlaylistExecutor) GetPlaylist(ctx context.Context, playlistID int) (*models.Playlist, error) {
	return e.repo.GetPlaylist(ctx, playlistID)
}

func (e *PlaylistExecutor) GetPlaylistItems(ctx context.Context, playlistID int) ([]*models.PlaylistItem, error) {
	return e.repo.GetPlaylistItems(ctx, playlistID)
}

func (e *PlaylistExecutor) GetMediaFiles(ctx context.Context, channelID int, page, pageSize int) ([]*models.MediaFile, error) {
	return e.repo.GetMediaFiles(ctx, channelID, page, pageSize)
}

func (e *PlaylistExecutor) CountMediaFiles(ctx context.Context, channelID int) (int, error) {
	return e.repo.CountMediaFiles(ctx, channelID)
}
