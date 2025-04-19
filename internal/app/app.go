package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/euacreations/tvheadend/internal/api"
	"github.com/euacreations/tvheadend/internal/config"
	"github.com/euacreations/tvheadend/internal/database"
	"github.com/euacreations/tvheadend/internal/services"
)

type Application struct {
	cfg    *config.Config
	repo   *database.Repository
	server *api.Server
}

func NewApplication(cfg *config.Config) (*Application, error) {
	repo, err := database.NewRepository(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	channelService := services.NewChannelService(repo)
	server := api.NewServer(channelService)

	return &Application{
		cfg:    cfg,
		repo:   repo,
		server: server,
	}, nil
}

func (a *Application) Start() error {
	log.Println("Starting TV Headend Server...")

	go func() {
		addr := ":" + strconv.Itoa(a.cfg.HTTPPort)
		log.Printf("HTTP server listening on %s", addr)
		if err := a.server.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	return nil
}

func (a *Application) Stop(ctx context.Context) error {
	log.Println("Shutting down server...")
	return a.repo.Close()
}
