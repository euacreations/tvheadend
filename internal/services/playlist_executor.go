package services

import (
	"context"
	"fmt"
	"log"
	"os"
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
	mediaCache map[int]*models.MediaFile
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
		mediaCache: make(map[int]*models.MediaFile),
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

			media, err := e.getMediaFile(ctx, currentItem.MediaID)
			if err != nil {
				return fmt.Errorf("media lookup failed: %w", err)
			}

			// Compute max duration for current item
			maxDuration := media.DurationSeconds
			if timeUntilTransition < time.Duration(media.DurationSeconds)*time.Second {
				maxDuration = int(timeUntilTransition.Seconds())
			}

			// --- Track FFmpeg Progress ---
			//go e.trackFFmpegProgress(ctx, channel)

			// --- Queue Next Item While Playing Current ---
			prepDone := make(chan struct{})
			go func() {
				defer close(prepDone)

				// Unlock current
				e.unlockItem(currentItem)

				// Re-fetch playlist if needed (optional - avoid if unchanged)
				items, err := e.repo.GetPlaylistItems(ctx, e.currentState.playlist.PlaylistID)
				if err == nil && len(items) > 0 {
					e.currentState.items = items
				}

				// Prepare next item
				nextIndex := (e.currentState.currentIndex + 1) % len(e.currentState.items)
				e.currentState.nextIndex = nextIndex
				e.lockItem(e.currentState.items[nextIndex])
			}()

			// --- Play Current Item (Blocking) ---
			err = e.playItem(ctx, channel, currentItem, e.currentState.startOffset, maxDuration)
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

			// Set current = next
			e.currentState.currentIndex = e.currentState.nextIndex
		}
	}
}

// func (e *PlaylistExecutor) trackFFmpegProgress(ctx context.Context, channel *models.Channel) {
// 	ticker := time.NewTicker(10 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-e.ffmpeg.Done():
// 			return
// 		case <-ticker.C:
// 			pos := parseFFmpegProgress(e.ffmpeg.logBuffer)
// 			//e.repo.UpdateChannelPosition(ctx, channel.ChannelID, pos)
// 		}
// 	}
// }

// var ffmpegTimeRegex = regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)

// func parseFFmpegProgress(logs string) float64 {
// 	lines := strings.Split(logs, "\n")

// 	// Search from bottom for latest time
// 	for i := len(lines) - 1; i >= 0; i-- {
// 		line := lines[i]
// 		if match := ffmpegTimeRegex.FindStringSubmatch(line); match != nil {
// 			if dur, err := time.ParseDuration(strings.Replace(match[1], ".", "s", 1) + "0ms"); err == nil {
// 				return dur.Seconds()
// 			}
// 		}
// 	}
// 	return 0
// }

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

	startIndex, startOffset := e.calculateStartPosition(ctx, items, effectiveDate, channel.StartTime)
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
	item *models.PlaylistItem, offset int, maxDuration int) error {

	//<-e.ffmpeg.Done()
	e.currentState.streamMux.Lock()
	defer e.currentState.streamMux.Unlock()

	// Reset the streamer before starting new stream
	e.ffmpeg.Reset()
	e.ffmpeg.SetProgressCallback(nil)

	media, err := e.getMediaFile(ctx, item.MediaID)
	if err != nil {
		return err
	}

	// Build FFmpeg config
	config := ffmpeg.StreamConfig{
		InputPath:               channel.StorageRoot + "media/" + media.FilePath,
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
			FilePath:  channel.StorageRoot + "data/" + overlay.FilePath,
			Text:      overlay.Text,
			PositionX: overlay.PositionX,
			PositionY: overlay.PositionY,
			FontSize:  overlay.FontSize,
			FontColor: overlay.FontColor,
		})
	}

	program_name_overlay := models.Overlay{
		Type:      "text",
		Text:      media.ProgramName,
		PositionX: "W/12",
		PositionY: "H/12",
		FontSize:  "H/45",
		FontColor: "white",
		FontFile:  channel.StorageRoot + "data/NotoSansSinhala-Regular.ttf",
	}

	// logo_overlay := models.Overlay{
	// 	Type:      "image",
	// 	Text:      media.ProgramName,
	// 	PositionX: "0.8*W",
	// 	PositionY: "H/12",
	// 	FontSize:  "H/45",
	// 	FontColor: "white",
	// 	FilePath:  channel.StorageRoot + "data/tvl_movies_logo_90.png",
	// }

	config.Overlays = append(config.Overlays, program_name_overlay)
	//config.Overlays = append(config.Overlays, logo_overlay)

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

		// Update DB (optional: add throttling if needed)
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
	//return nil
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

func (e *PlaylistExecutor) getMediaFile(ctx context.Context, mediaID int) (*models.MediaFile, error) {
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
	playlistStart time.Time, channelStart time.Time) (int, int) {

	elapsed := time.Since(playlistStart)
	if elapsed < 0 {
		return 0, 0
	}

	totalDuration := 0
	for _, item := range items {
		media, err := e.getMediaFile(ctx, item.MediaID)
		if err != nil {
			return 0, 0
		}

		totalDuration += media.DurationSeconds
	}

	positionSec := int(elapsed.Seconds()) % (24 * 3600)
	if totalDuration > 0 {
		positionSec %= totalDuration
	}

	accumulated := 0
	for i, item := range items {
		media, err := e.getMediaFile(ctx, item.MediaID)
		if err != nil {
			return 0, 0
		}

		if accumulated+media.DurationSeconds > positionSec {
			return i, positionSec - accumulated
		}
		accumulated += media.DurationSeconds
	}

	return 0, 0
}

func (e *PlaylistExecutor) GetPlaylists(ctx context.Context, channelID int) ([]*models.Playlist, error) {
	playlist, err := e.repo.GetPlaylists(ctx, channelID)
	if err != nil {
		fmt.Errorf("failed to get playlists: %w", err)
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
