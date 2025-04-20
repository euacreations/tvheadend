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

type ChannelService struct {
	repo      *database.Repository
	streamers map[int]*ffmpeg.Streamer
	streamMux sync.Mutex
}

func NewChannelService(repo *database.Repository) *ChannelService {
	return &ChannelService{
		repo:      repo,
		streamers: make(map[int]*ffmpeg.Streamer),
	}
}

func (s *ChannelService) GetAllChannels(ctx context.Context) ([]*models.Channel, error) {
	channels, err := s.repo.GetAllChannels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve channels: %w", err)
	}

	// Get states for all channels
	for _, channel := range channels {
		state, err := s.repo.GetChannelState(ctx, channel.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get state for channel %d: %w", channel.ID, err)
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
	config := ffmpeg.StreamConfig{
		InputPath:    channel.StorageRoot + "/current_stream.ts",
		OutputURL:    channel.OutputUDP,
		VideoCodec:   "hevc_nvenc",
		VideoBitrate: "800k",
		MinBitrate:   "800k",
		MaxBitrate:   "800k",
		BufferSize:   "1600k",
	}

	if err := streamer.Start(ctx, config); err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	s.streamers[channelID] = streamer

	state := &models.ChannelState{
		ChannelID:      channelID,
		Running:        true,
		FFmpegPID:      streamer.PID(),
		LastUpdateTime: time.Now(),
	}

	if err := s.repo.UpdateChannelState(ctx, state); err != nil {
		_ = streamer.Stop()
		delete(s.streamers, channelID)
		return fmt.Errorf("failed to update channel state: %w", err)
	}

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
