package app

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/euacreations/tvheadend/internal/api"
	"github.com/euacreations/tvheadend/internal/config"
	"github.com/euacreations/tvheadend/internal/database"
	"github.com/euacreations/tvheadend/internal/services"
	"github.com/euacreations/tvheadend/pkg/ffmpeg"
)

type Application struct {
	cfg            *config.Config
	repo           *database.Repository
	server         *api.Server
	channelService *services.ChannelService
	mediaScanner   *services.MediaScanner
	playlistExec   *services.PlaylistExecutor
}

func NewApplication(cfg *config.Config) (*Application, error) {
	repo, err := database.NewRepository(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	ffmpeg := ffmpeg.New()
	channelService := services.NewChannelService(repo)
	mediaScanner := services.NewMediaScanner(repo)
	playlistExec := services.NewPlaylistExecutor(repo, ffmpeg)
	overlayService := services.NewOverlayService(repo)

	server := api.NewServer(channelService, mediaScanner, playlistExec, overlayService)

	return &Application{
		cfg:            cfg,
		repo:           repo,
		server:         server,
		channelService: channelService,
		mediaScanner:   mediaScanner,
		playlistExec:   playlistExec,
	}, nil
}

func (a *Application) Start() error {
	// Start background services
	go a.startBackgroundServices()

	// Start HTTP server

	// Start all enabled channels
	ctx := context.Background()
	channels, err := a.channelService.GetAllChannels(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve channels: %w", err)
	}

	for _, ch := range channels {
		if ch.Enabled {
			go func(channelID int) {
				if err := a.channelService.StartChannel(ctx, channelID); err != nil {
					log.Printf("Failed to start channel %d: %v", channelID, err)
				} else {
					log.Printf("Channel %d started", channelID)
				}
			}(ch.ChannelID)
		}
	}

	return a.server.Start(":" + strconv.Itoa(a.cfg.HTTPPort))

}

func (a *Application) startBackgroundServices() {
	// Scan media files periodically
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		channels, err := a.repo.GetAllChannels(context.Background())
		if err != nil {
			log.Printf("Failed to get channels for scanning: %v", err)
			continue
		}

		for _, channel := range channels {
			if err := a.mediaScanner.ScanChannelMedia(context.Background(), channel.ChannelID); err != nil {
				log.Printf("Failed to scan media for channel %d: %v", channel.ChannelID, err)
			}
		}
	}
}

func (a *Application) Stop(ctx context.Context) error {
	log.Println("Shutting down server...")
	return a.repo.Close()
}
