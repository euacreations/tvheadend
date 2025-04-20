package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/euacreations/tvheadend/internal/config"
	"github.com/euacreations/tvheadend/internal/models"
	_ "github.com/go-sql-driver/mysql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(cfg *config.Config) (*Repository, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := sql.Open("mysql", dsn)
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

func (r *Repository) GetChannelByID(ctx context.Context, id int) (*models.Channel, error) {
	query := `SELECT channel_id, channel_name, storage_root, output_udp, playlist_type, 
		start_time, enabled, use_previous_day_fallback, created_at, updated_at 
		FROM channels WHERE channel_id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	var channel models.Channel
	var startTime string

	err := row.Scan(
		&channel.ID,
		&channel.Name,
		&channel.StorageRoot,
		&channel.OutputUDP,
		&channel.PlaylistType,
		&startTime,
		&channel.Enabled,
		&channel.UsePreviousDayFallback,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan channel: %w", err)
	}

	channel.StartTime, err = time.Parse("15:04:05", startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start time: %w", err)
	}

	return &channel, nil
}

func (r *Repository) GetAllChannels(ctx context.Context) ([]*models.Channel, error) {
	query := `SELECT 
        channel_id, channel_name, storage_root, output_udp, playlist_type, 
        start_time, enabled, use_previous_day_fallback, created_at, updated_at 
        FROM channels
        ORDER BY channel_id` // Added ordering for consistent results

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	var channels []*models.Channel

	for rows.Next() {
		var channel models.Channel
		var startTime string

		if err := rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.StorageRoot,
			&channel.OutputUDP,
			&channel.PlaylistType,
			&startTime,
			&channel.Enabled,
			&channel.UsePreviousDayFallback,
			&channel.CreatedAt,
			&channel.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}

		// Parse start time with error handling
		if channel.StartTime, err = time.Parse("15:04:05", startTime); err != nil {
			return nil, fmt.Errorf("failed to parse start time '%s': %w", startTime, err)
		}

		channels = append(channels, &channel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return channels, nil
}

func (r *Repository) GetChannelState(ctx context.Context, channelID int) (*models.ChannelState, error) {
	query := `SELECT 
        channel_id, current_playlist_id, current_item_id, 
        current_position_seconds, running, ffmpeg_pid, last_update_time
        FROM channel_states 
        WHERE channel_id = ?`

	row := r.db.QueryRowContext(ctx, query, channelID)

	var state models.ChannelState
	err := row.Scan(
		&state.ChannelID,
		&state.CurrentPlaylistID,
		&state.CurrentItemID,
		&state.CurrentPosition,
		&state.Running,
		&state.FFmpegPID,
		&state.LastUpdateTime,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return empty state if not found
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
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
		current_playlist_id = VALUES(current_playlist_id),
		current_item_id = VALUES(current_item_id),
		current_position_seconds = VALUES(current_position_seconds),
		running = VALUES(running),
		ffmpeg_pid = VALUES(ffmpeg_pid),
		last_update_time = VALUES(last_update_time)`

	_, err := r.db.ExecContext(ctx, query,
		state.ChannelID,
		state.CurrentPlaylistID,
		state.CurrentItemID,
		state.CurrentPosition,
		state.Running,
		state.FFmpegPID,
		state.LastUpdateTime,
	)
	return err
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

func (r *Repository) GetMediaFile(ctx context.Context, mediaID int) (*models.MediaFile, error) {
	query := `SELECT id, channel_id, file_path, file_name, duration_seconds, 
              program_name, file_size, last_modified, scanned_at, created_at, updated_at
              FROM media_files WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, mediaID)
	var mf models.MediaFile
	err := row.Scan(
		&mf.ID,
		&mf.ChannelID,
		&mf.FilePath,
		&mf.FileName,
		&mf.DurationSeconds,
		&mf.ProgramName,
		&mf.FileSize,
		&mf.LastModified,
		&mf.ScannedAt,
		&mf.CreatedAt,
		&mf.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &mf, nil
}

func (r *Repository) GetPlaylist(ctx context.Context, playlistID int) (*models.Playlist, error) {
	query := `SELECT 
        playlist_id, channel_id, playlist_date, status, 
        total_duration_seconds, created_at, updated_at
        FROM playlists 
        WHERE playlist_id = ?`

	row := r.db.QueryRowContext(ctx, query, playlistID)
	var p models.Playlist
	err := row.Scan(
		&p.ID,
		&p.ChannelID,
		&p.PlaylistDate,
		&p.Status,
		&p.TotalDurationSeconds,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) GetActivePlaylist(ctx context.Context, channelID int) (*models.Playlist, error) {
	query := `SELECT id, channel_id, playlist_date, status, total_duration_seconds, 
              created_at, updated_at 
              FROM playlists 
              WHERE channel_id = ? AND status = 'active' 
              ORDER BY created_at DESC LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, channelID)
	var p models.Playlist
	err := row.Scan(
		&p.ID,
		&p.ChannelID,
		&p.PlaylistDate,
		&p.Status,
		&p.TotalDurationSeconds,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) GetPlaylistItems(ctx context.Context, playlistID int) ([]*models.PlaylistItem, error) {
	query := `SELECT id, playlist_id, media_id, position, 
              scheduled_start_time, scheduled_end_time,
              actual_start_time, actual_end_time, locked,
              created_at, updated_at
              FROM playlist_items 
              WHERE playlist_id = ?
              ORDER BY position`

	rows, err := r.db.QueryContext(ctx, query, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.PlaylistItem
	for rows.Next() {
		var item models.PlaylistItem
		err := rows.Scan(
			&item.ID,
			&item.PlaylistID,
			&item.MediaID,
			&item.Position,
			&item.ScheduledStartTime,
			&item.ScheduledEndTime,
			&item.ActualStartTime,
			&item.ActualEndTime,
			&item.Locked,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) GetChannelOverlays(ctx context.Context, channelID int) ([]*models.Overlay, error) {
	query := `SELECT id, channel_id, type, file_path, 
              position_x, position_y,
              enabled, font_size, font_color,
			  created_at, updated_at
              FROM overlays 
              WHERE channel_id = ? AND enabled = TRUE`

	rows, err := r.db.QueryContext(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var overlays []*models.Overlay
	for rows.Next() {
		var o models.Overlay
		err := rows.Scan(
			&o.ID,
			&o.ChannelID,
			&o.Type,
			&o.FilePath,
			&o.PositionX,
			&o.PositionY,
			&o.Enabled,
			&o.FontSize,
			&o.FontColor,
			&o.CreatedAt,
			&o.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		overlays = append(overlays, &o)
	}

	if err := rows.Err(); err != nil {
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
		&overlay.ID,
		&overlay.CreatedAt,
		&overlay.UpdatedAt,
	)
}
