package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/euacreations/tvheadend/internal/database"
	"github.com/euacreations/tvheadend/internal/models"
)

type MediaScanner struct {
	repo *database.Repository
}

func NewMediaScanner(repo *database.Repository) *MediaScanner {
	return &MediaScanner{repo: repo}
}

func (s *MediaScanner) ScanChannelMedia(ctx context.Context, channelID int) error {
	channel, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	mediaDir := filepath.Join(channel.StorageRoot, "media")
	files, err := os.ReadDir(mediaDir)
	if err != nil {
		return fmt.Errorf("failed to read media directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(mediaDir, file.Name())
		fileInfo, err := file.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		// Check if file already exists in database
		exists, err := s.repo.MediaFileExists(ctx, channelID, filePath)
		if err != nil {
			return err
		}

		if !exists {
			// In production, you'd use ffprobe to get duration
			mediaFile := models.MediaFile{
				ChannelID:    channelID,
				FilePath:     filePath,
				FileName:     file.Name(),
				FileSize:     fileInfo.Size(),
				LastModified: fileInfo.ModTime(),
				ScannedAt:    time.Now(),
			}

			if err := s.repo.CreateMediaFile(ctx, &mediaFile); err != nil {
				return err
			}
		}
	}

	return nil
}
