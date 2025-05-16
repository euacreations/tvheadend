package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/euacreations/tvheadend/internal/database"
	"github.com/euacreations/tvheadend/internal/models"
	"github.com/euacreations/tvheadend/pkg/ffmpeg"
)

type ChannelService struct {
	repo             *database.Repository
	playlistExecutor *PlaylistExecutor
	streamers        map[int]*ffmpeg.Streamer
	streamMux        sync.Mutex
}

func NewChannelService(repo *database.Repository) *ChannelService {
	return &ChannelService{
		repo:             repo,
		streamers:        make(map[int]*ffmpeg.Streamer),
		playlistExecutor: NewPlaylistExecutor(repo, ffmpeg.New()),
	}
}

func (s *ChannelService) GetAllChannels(ctx context.Context) ([]*models.Channel, error) {
	channels, err := s.repo.GetAllChannels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve channels: %w", err)
	}
	// Get states for all channels
	for _, channel := range channels {
		state, err := s.repo.GetChannelState(ctx, channel.ChannelID)

		if err != nil {
			return nil, fmt.Errorf("failed to get state for channel %d: %w", channel.ChannelID, err)
		}
		channel.State = state
	}

	return channels, nil
}

func (s *ChannelService) GetChannel(ctx context.Context, id int) (*models.Channel, error) {
	channel, err := s.repo.GetChannelByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	state, err := s.repo.GetChannelState(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel state: %w", err)
	}
	channel.State = state

	return channel, nil
}

func (s *ChannelService) CheckChannelStatus(ctx context.Context, channelID int) (bool, error) {
	s.streamMux.Lock()
	defer s.streamMux.Unlock()

	_, exists := s.streamers[channelID]
	return exists, nil
}

func (s *ChannelService) StartChannel(ctx context.Context, channelID int) error {
	channel, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	s.streamMux.Lock()
	defer s.streamMux.Unlock()
	if _, exists := s.streamers[channelID]; exists {
		return fmt.Errorf("channel %d is already running", channelID)
	}

	streamer := ffmpeg.New()
	s.streamers[channelID] = streamer

	streamer.SetProgressCallback(func(position float64) {
		state := &models.ChannelState{
			ChannelID:       channelID,
			Running:         true,
			CurrentPosition: position,
			LastUpdateTime:  time.Now(),
			FFmpegPID:       streamer.PID(),
		}
		if err := s.repo.UpdateChannelState(context.Background(), state); err != nil {
			log.Printf("Failed to update channel state: %v", err)
		}
	})

	switch channel.PlaylistType {
	case "daily_playlist":
		executor := NewPlaylistExecutor(s.repo, streamer)

		go func() {
			if err := executor.Execute(context.Background(), channel); err != nil {
				log.Printf("PlaylistExecutor error: %v", err)
				// Clean up on error
				s.streamMux.Lock()
				delete(s.streamers, channelID)
				s.streamMux.Unlock()
			}
		}()

	default:
		// Default behavior (existing code)
		return s.startDefaultStream(ctx, channel)
	}

	return nil

}

func (s *ChannelService) startDefaultStream(ctx context.Context, channel *models.Channel) error {
	// Existing code to start a single stream
	return nil
}

func (s *ChannelService) StopChannel(ctx context.Context, channelID int) error {
	s.streamMux.Lock()
	streamer, exists := s.streamers[channelID]
	s.streamMux.Unlock()

	if !exists {
		return fmt.Errorf("channel %d is not running", channelID)
	}

	if err := streamer.Stop(); err != nil {
		return fmt.Errorf("failed to stop stream: %w", err)
	}

	state := &models.ChannelState{
		ChannelID:      channelID,
		Running:        false,
		FFmpegPID:      0,
		LastUpdateTime: time.Now(),
	}

	if err := s.repo.UpdateChannelState(ctx, state); err != nil {
		return fmt.Errorf("failed to update channel state: %w", err)
	}

	s.streamMux.Lock()
	delete(s.streamers, channelID)
	s.streamMux.Unlock()

	return nil
}
