package database

import (
	"context"
	"fmt"
	"time"

	"github.com/euacreations/tvheadend/internal/config"
	"github.com/euacreations/tvheadend/internal/models"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(cfg *config.Config) (*Repository, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) GetChannelByID(ctx context.Context, ChannelID int) (*models.Channel, error) {
	query := `SELECT * FROM channels WHERE channel_id = ?`
	var channel models.Channel
	err := r.db.GetContext(ctx, &channel, query, ChannelID)

	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}
	channel.StartTime, err = time.Parse("15:04:05", channel.StartTimeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start_time for channel %d: %w", channel.ChannelID, err)
	}
	return &channel, nil
}

func (r *Repository) GetAllChannels(ctx context.Context) ([]*models.Channel, error) {
	query := `SELECT * FROM channels ORDER BY channel_id`
	var channels []*models.Channel
	err := r.db.SelectContext(ctx, &channels, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)

	}

	for _, channel := range channels {
		channel.StartTime, err = time.Parse("15:04:05", channel.StartTimeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start_time for channel %d: %w", channel.ChannelID, err)
		}
	}

	return channels, nil
}

func (r *Repository) GetChannelState(ctx context.Context, channelID int) (*models.ChannelState, error) {
	query := `SELECT * FROM channel_states WHERE channel_id = ?`
	var state models.ChannelState
	err := r.db.GetContext(ctx, &state, query, channelID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return &models.ChannelState{
				ChannelID: channelID,
				Running:   false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get channel state: %w", err)
	}

	return &state, nil
}

func (r *Repository) UpdateChannelState(ctx context.Context, state *models.ChannelState) error {
	query := `INSERT INTO channel_states 
        (channel_id, current_playlist_id, current_item_id, current_position_seconds, 
        running, ffmpeg_pid, last_update_time) 
        VALUES (:channel_id, :current_playlist_id, :current_item_id, :current_position_seconds, 
        :running, :ffmpeg_pid, :last_update_time)
        ON DUPLICATE KEY UPDATE 
        current_playlist_id = VALUES(current_playlist_id),
        current_item_id = VALUES(current_item_id),
        current_position_seconds = VALUES(current_position_seconds),
        running = VALUES(running),
        ffmpeg_pid = VALUES(ffmpeg_pid),
        last_update_time = VALUES(last_update_time)`

	// Use NamedExecContext to automatically map struct fields to named parameters
	_, err := r.db.NamedExecContext(ctx, query, state)
	if err != nil {
		return fmt.Errorf("failed to update channel state: %w", err)
	}

	return nil
}

func (r *Repository) MediaFileExists(ctx context.Context, channelID int, filePath string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM media_files WHERE channel_id = ? AND file_path = ?)`
	err := r.db.QueryRowContext(ctx, query, channelID, filePath).Scan(&exists)
	return exists, err
}

func (r *Repository) CreateMediaFile(ctx context.Context, file *models.MediaFile) error {
	query := `INSERT INTO media_files 
		(channel_id, file_path, file_name, duration_seconds, file_size, last_modified, scanned_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		file.ChannelID,
		file.FilePath,
		file.FileName,
		file.DurationSeconds,
		file.FileSize,
		file.LastModified,
		file.ScannedAt,
	)
	return err
}

func (r *Repository) GetMediaFiles(ctx context.Context, channelID int) ([]*models.MediaFile, error) {
	query := `SELECT media_id, channel_id, file_path, file_name, duration_seconds, 
            program_name, file_size, last_modified, scanned_at, created_at, updated_at
			FROM media_files WHERE channel_id = ?`

	var mf []*models.MediaFile
	err := r.db.SelectContext(ctx, &mf, query, channelID)
	if err != nil {
		return nil, err
	}
	return mf, nil

}

func (r *Repository) GetMediaFile(ctx context.Context, mediaID int) (*models.MediaFile, error) {
	query := `SELECT media_id, channel_id, file_path, file_name, duration_seconds, 
            program_name, file_size, last_modified, scanned_at, created_at, updated_at
			FROM media_files WHERE media_id = ?`

	var mf *models.MediaFile
	err := r.db.SelectContext(ctx, &mf, query, mediaID)
	if err != nil {
		return nil, err
	}
	return mf, nil

}

func (r *Repository) GetPlaylists(ctx context.Context, channelID int) ([]*models.Playlist, error) {
	query := `SELECT 
		playlist_id, channel_id, playlist_date, status, 
		total_duration_seconds, created_at, updated_at
		FROM playlists
		WHERE channel_id = ?`

	var playlists []*models.Playlist
	err := r.db.SelectContext(ctx, &playlists, query, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlists: %w", err)
	}

	return playlists, nil
}

func (r *Repository) GetPlaylist(ctx context.Context, playlistID int) (*models.Playlist, error) {
	query := `SELECT 
		playlist_id, channel_id, playlist_date, status, 
		total_duration_seconds, created_at, updated_at
		FROM playlists
		WHERE playlist_id = ?`

	var playlist *models.Playlist
	err := r.db.SelectContext(ctx, &playlist, query, playlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlists: %w", err)
	}

	return playlist, nil
}

func (r *Repository) GetActivePlaylist(ctx context.Context, channelID int) (*models.Playlist, error) {
	query := `SELECT * FROM playlists 
              WHERE channel_id = ? AND status = 'active' 
              ORDER BY created_at DESC LIMIT 1`

	var playlist *models.Playlist
	err := r.db.SelectContext(ctx, &playlist, query, channelID)

	if err != nil {
		return nil, err
	}
	return playlist, nil
}

func (r *Repository) GetPlaylistItems(ctx context.Context, playlistID int) ([]*models.PlaylistItem, error) {
	query := `SELECT * FROM playlist_items WHERE playlist_id = ? ORDER BY position`

	var items []*models.PlaylistItem
	err := r.db.SelectContext(ctx, &items, query, playlistID)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) GetChannelOverlays(ctx context.Context, channelID int) ([]*models.Overlay, error) {
	query := `SELECT * FROM overlays WHERE channel_id = ? AND enabled = TRUE`

	var overlays []*models.Overlay
	err := r.db.SelectContext(ctx, &overlays, query, channelID)

	if err != nil {
		return nil, err
	}
	return overlays, nil
}

func (r *Repository) CreateOverlay(ctx context.Context, overlay *models.Overlay) error {
	query := `INSERT INTO overlays 
        (channel_id, type, file_path, position_x, position_y, 
         enabled, font_size, font_color)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        RETURNING id, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, // Note: lowercase r.db if not exported
		overlay.ChannelID,
		overlay.Type,
		overlay.FilePath,
		overlay.PositionX,
		overlay.PositionY,
		overlay.Enabled,
		overlay.FontSize,
		overlay.FontColor,
	)

	return row.Scan(
		&overlay.OverlayID,
		&overlay.CreatedAt,
		&overlay.UpdatedAt,
	)
}
