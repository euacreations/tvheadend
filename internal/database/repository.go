package database

import (
	"context"
	"database/sql"
	"fmt"
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
