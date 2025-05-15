package services

import (
	"context"
	"fmt"

	"github.com/euacreations/tvheadend/internal/database"
	"github.com/euacreations/tvheadend/internal/models"
)

type OverlayService struct {
	repo *database.Repository
}

func NewOverlayService(repo *database.Repository) *OverlayService {
	return &OverlayService{repo: repo}
}

func (s *OverlayService) ApplyOverlays(ctx context.Context, channelID int, ffmpegArgs []string) ([]string, error) {
	overlays, err := s.repo.GetChannelOverlays(ctx, channelID)

	if err != nil {
		return nil, fmt.Errorf("failed to get overlays: %w", err)
	}

	for _, overlay := range overlays {
		if overlay.Enabled {
			switch overlay.Type {
			case "image":
				ffmpegArgs = append(ffmpegArgs, buildImageOverlayArgs(overlay)...)
			case "text":
				ffmpegArgs = append(ffmpegArgs, buildTextOverlayArgs(overlay, s.getDynamicText(ctx, channelID))...)
			}
		}
	}

	return ffmpegArgs, nil
}

func buildImageOverlayArgs(overlay *models.Overlay) []string {
	return []string{
		"-i", overlay.FilePath,
		//"-filter_complex", fmt.Sprintf("[0:v][1:v]overlay=%d:%d", overlay.PositionX, overlay.PositionY),
	}
}

func buildTextOverlayArgs(overlay *models.Overlay, text string) []string {
	return []string{
		"-vf", fmt.Sprintf("drawtext=text='%s':x=%s:y=%s:fontsize=%s:fontcolor=%s",
			text, overlay.PositionX, overlay.PositionY, overlay.FontSize, overlay.FontColor),
	}
}

func (s *OverlayService) getDynamicText(ctx context.Context, channelID int) string {
	// Implementation for dynamic text generation
	return "Sample Text"
}

func (s *OverlayService) CreateOverlay(ctx context.Context, overlay *models.Overlay) (*models.Overlay, error) {
	// Validate input
	if overlay.ChannelID == 0 {
		return nil, fmt.Errorf("channel ID is required")
	}
	if overlay.Type != "image" && overlay.Type != "text" {
		return nil, fmt.Errorf("invalid overlay type")
	}
	if overlay.Type == "image" && overlay.FilePath == "" {
		return nil, fmt.Errorf("file path is required for image overlays")
	}

	// Set defaults
	if overlay.PositionX == "" && overlay.PositionY == "" {
		overlay.PositionX = "10"
		overlay.PositionY = "10"
	}
	if overlay.FontSize == "" {
		overlay.FontSize = "24"
	}
	if overlay.FontColor == "" {
		overlay.FontColor = "white"
	}

	if err := s.repo.CreateOverlay(ctx, overlay); err != nil {
		return nil, fmt.Errorf("failed to create overlay: %w", err)
	}
	return overlay, nil
}
