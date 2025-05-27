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
	executorCancels  map[int]context.CancelFunc
	streamMux        sync.Mutex
}

func NewChannelService(repo *database.Repository) *ChannelService {
	return &ChannelService{
		repo:             repo,
		streamers:        make(map[int]*ffmpeg.Streamer),
		executorCancels:  make(map[int]context.CancelFunc),
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

func (s *ChannelService) GetChannelStatus(ctx context.Context, channelID int) (*models.ChannelState, bool, error) {
	// Get the current state from the database
	state, err := s.repo.GetChannelState(ctx, channelID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get channel state: %w", err)
	}

	// Check if the streamer is actually running in our application memory
	s.streamMux.Lock()
	streamer, isRunning := s.streamers[channelID]
	s.streamMux.Unlock()

	// If we have a streamer but the state says not running, update the state
	if isRunning && !state.Running {
		state.Running = true
		state.FFmpegPID = streamer.PID()
		state.LastUpdateTime = time.Now()

		if err := s.repo.UpdateChannelState(ctx, state); err != nil {
			return state, isRunning, fmt.Errorf("failed to update channel state: %w", err)
		}
	}

	// If we don't have a streamer but state says running, update the state
	if !isRunning && state.Running {
		state.Running = false
		state.FFmpegPID = 0
		state.LastUpdateTime = time.Now()

		if err := s.repo.UpdateChannelState(ctx, state); err != nil {
			return state, isRunning, fmt.Errorf("failed to update channel state: %w", err)
		}
	}

	return state, isRunning, nil
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
		executorCtx, cancel := context.WithCancel(context.Background())
		s.executorCancels[channelID] = cancel

		go func() {

			defer func() {
				// Clean up on exit
				s.streamMux.Lock()
				delete(s.streamers, channelID)
				delete(s.executorCancels, channelID)
				s.streamMux.Unlock()
			}()

			if err := executor.Execute(executorCtx, channel); err != nil {
				// Only log error if it's not due to context cancellation
				if err != context.Canceled {
					log.Printf("PlaylistExecutor error: %v", err)
				}
			}
			// if err := executor.Execute(context.Background(), channel); err != nil {
			// 	log.Printf("PlaylistExecutor error: %v", err)
			// 	// Clean up on error
			// 	s.streamMux.Lock()
			// 	delete(s.streamers, channelID)
			// 	s.streamMux.Unlock()
			// }
		}()

	default:
		// Default behavior (existing code)
		return s.startDefaultStream(ctx, channel)
		//return nil
	}

	return nil

}

func (s *ChannelService) startDefaultStream(ctx context.Context, channel *models.Channel) error {
	// Existing code to start a single stream
	return nil
}

func (s *ChannelService) StopChannel(ctx context.Context, channelID int) error {
	s.streamMux.Lock()
	streamer, streamerExists := s.streamers[channelID]
	cancel, cancelExists := s.executorCancels[channelID]
	s.streamMux.Unlock()

	if !streamerExists {
		return fmt.Errorf("channel %d is not running", channelID)
	}

	streamer.SetProgressCallback(nil) // Disable further callbacks

	if cancelExists {
		cancel()
	}

	if err := streamer.Stop(); err != nil {
		return fmt.Errorf("failed to stop stream: %w", err)
	}

	// Retrieve the current state before modifying
	currentState, err := s.repo.GetChannelState(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get current channel state: %w", err)
	}

	currentState.Running = false
	currentState.FFmpegPID = 0
	currentState.LastUpdateTime = time.Now()

	// Update the channel state in the database
	if err := s.repo.UpdateChannelState(ctx, currentState); err != nil {
		return fmt.Errorf("failed to update channel state: %w", err)
	}

	s.streamMux.Lock()
	delete(s.streamers, channelID)
	delete(s.executorCancels, channelID)
	s.streamMux.Unlock()

	return nil
}
