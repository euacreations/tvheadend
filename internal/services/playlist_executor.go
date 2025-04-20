package services

import (
	"context"
	"fmt"
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
}

func NewPlaylistExecutor(repo *database.Repository, ffmpeg *ffmpeg.Streamer) *PlaylistExecutor {
	return &PlaylistExecutor{
		repo:       repo,
		ffmpeg:     ffmpeg,
		mediaCache: make(map[int]*models.MediaFile),
	}
}

func (e *PlaylistExecutor) GetPlaylist(ctx context.Context, playlistID int) (*models.Playlist, error) {
	return e.repo.GetPlaylist(ctx, playlistID)
}

func (e *PlaylistExecutor) GetPlaylistItems(ctx context.Context, playlistID int) ([]*models.PlaylistItem, error) {
	return e.repo.GetPlaylistItems(ctx, playlistID)
}

func (e *PlaylistExecutor) Execute(ctx context.Context, channelID int) error {
	playlist, err := e.repo.GetActivePlaylist(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get active playlist: %w", err)
	}

	items, err := e.GetPlaylistItems(ctx, playlist.ID)
	if err != nil {
		return fmt.Errorf("failed to get playlist items: %w", err)
	}

	for _, item := range items {
		if err := e.executeItem(ctx, channelID, item); err != nil {
			return err
		}
	}

	return nil
}

func (e *PlaylistExecutor) executeItem(ctx context.Context, channelID int, item *models.PlaylistItem) error {
	media, err := e.getMediaFile(ctx, item.MediaID)
	if err != nil {
		return fmt.Errorf("failed to get media file: %w", err)
	}

	// Update playback state
	state := &models.ChannelState{
		ChannelID:         channelID,
		CurrentPlaylistID: item.PlaylistID,
		CurrentItemID:     item.ID,
		CurrentPosition:   0,
		Running:           true,
		LastUpdateTime:    time.Now(),
	}

	if err := e.repo.UpdateChannelState(ctx, state); err != nil {
		return fmt.Errorf("failed to update channel state: %w", err)
	}

	// Update state with playback position in real-time
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state.CurrentPosition += 1.0
				_ = e.repo.UpdateChannelState(ctx, state) // Best effort update
			}
		}
	}()

	// Get channel for output config
	channel, err := e.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Execute FFmpeg stream
	config := ffmpeg.StreamConfig{
		InputPath:    media.FilePath,
		OutputURL:    channel.OutputUDP,
		VideoCodec:   "hevc_nvenc",
		VideoBitrate: "800k",
	}

	return e.ffmpeg.Start(ctx, config)
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
