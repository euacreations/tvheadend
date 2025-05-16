package database

import (
	"context"
	"database/sql"
	"errors"
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Asia%%2FColombo",
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

func (r *Repository) UpdateChannel(ctx context.Context, channel *models.Channel) error {
	query := `
		INSERT INTO channel (
			channel_id,
			channel_name,
			storage_root,
			output_udp,
			playlist_type,
			playlist_id,
			start_time,
			enabled,
			use_previous_day_fallback,
			video_codec,
			video_bitrate,
			min_bitrate,
			max_bitrate,
			audio_codec,
			audio_bitrate,
			buffer_size,
			packet_size,
			output_resolution,
			mpegts_original_network_id,
			mpegts_transport_stream_id,
			mpegts_service_id,
			mpegts_start_pid,
			mpegts_pmt_start_pid,
			metadata_service_provider,
			created_at,
			updated_at
		) VALUES (
			:channel_id,
			:channel_name,
			:storage_root,
			:output_udp,
			:playlist_type,
			:playlist_id,
			:start_time,
			:enabled,
			:use_previous_day_fallback,
			:video_codec,
			:video_bitrate,
			:min_bitrate,
			:max_bitrate,
			:audio_codec,
			:audio_bitrate,
			:buffer_size,
			:packet_size,
			:output_resolution,
			:mpegts_original_network_id,
			:mpegts_transport_stream_id,
			:mpegts_service_id,
			:mpegts_start_pid,
			:mpegts_pmt_start_pid,
			:metadata_service_provider,
			NOW(),
			NOW()
		)
		ON DUPLICATE KEY UPDATE
			channel_name = VALUES(channel_name),
			storage_root = VALUES(storage_root),
			output_udp = VALUES(output_udp),
			playlist_type = VALUES(playlist_type),
			playlist_id = VALUES(playlist_id),
			start_time = VALUES(start_time),
			enabled = VALUES(enabled),
			use_previous_day_fallback = VALUES(use_previous_day_fallback),
			video_codec = VALUES(video_codec),
			video_bitrate = VALUES(video_bitrate),
			min_bitrate = VALUES(min_bitrate),
			max_bitrate = VALUES(max_bitrate),
			audio_codec = VALUES(audio_codec),
			audio_bitrate = VALUES(audio_bitrate),
			buffer_size = VALUES(buffer_size),
			packet_size = VALUES(packet_size),
			output_resolution = VALUES(output_resolution),
			mpegts_original_network_id = VALUES(mpegts_original_network_id),
			mpegts_transport_stream_id = VALUES(mpegts_transport_stream_id),
			mpegts_service_id = VALUES(mpegts_service_id),
			mpegts_start_pid = VALUES(mpegts_start_pid),
			mpegts_pmt_start_pid = VALUES(mpegts_pmt_start_pid),
			metadata_service_provider = VALUES(metadata_service_provider),
			updated_at = NOW()
	`

	// NamedExecContext uses struct field tags (db:"...") for parameter binding
	_, err := r.db.NamedExecContext(ctx, query, channel)
	if err != nil {
		return fmt.Errorf("failed to insert/update channel: %w", err)
	}
	return nil
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
        (channel_id, current_playlist_id, current_item_id, current_position, 
        running, ffmpeg_pid, last_update_time) 
        VALUES (:channel_id, :current_playlist_id, :current_item_id, :current_position, 
        :running, :ffmpeg_pid, :last_update_time)
        ON DUPLICATE KEY UPDATE 
        current_playlist_id = VALUES(current_playlist_id),
        current_item_id = VALUES(current_item_id),
        current_position = VALUES(current_position),
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

/*func (r *Repository) GetMediaFiles(ctx context.Context, channelID int) ([]*models.MediaFile, error) {
	query := `SELECT media_id, channel_id, file_path, file_name, duration_seconds,
            program_name, file_size, last_modified, scanned_at, created_at, updated_at
			FROM media_files WHERE channel_id = ?`

	var mf []*models.MediaFile
	err := r.db.SelectContext(ctx, &mf, query, channelID)
	if err != nil {
		return nil, err
	}
	return mf, nil

}*/

func (r *Repository) GetMediaFiles(ctx context.Context, channelID int, page, pageSize int) ([]*models.MediaFile, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10 // Default page size
	}

	offset := (page - 1) * pageSize

	query := `SELECT media_id, channel_id, file_path, file_name, duration_seconds, 
            program_name, file_size, last_modified, scanned_at, created_at, updated_at
            FROM media_files 
            WHERE channel_id = ?
            LIMIT ? OFFSET ?`

	var mf []*models.MediaFile
	err := r.db.SelectContext(ctx, &mf, query, channelID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return mf, nil
}

func (r *Repository) CountMediaFiles(ctx context.Context, channelID int) (int, error) {
	query := `SELECT COUNT(*) FROM media_files WHERE channel_id = ?`

	var count int
	err := r.db.GetContext(ctx, &count, query, channelID)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) GetMediaFile(ctx context.Context, mediaID sql.NullInt64) (*models.MediaFile, error) {
	query := `SELECT media_id, channel_id, file_path, file_name, duration_seconds, 
            program_name, file_size, last_modified, scanned_at, created_at, updated_at
			FROM media_files WHERE media_id = ?`

	var mf models.MediaFile
	err := r.db.GetContext(ctx, &mf, query, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file with ID %d: %w", mediaID.Int64, err)
	}
	return &mf, nil
}

func (r *Repository) GetUDPStream(ctx context.Context, streamID sql.NullInt64) (*models.UDPStream, error) {
	query := `SELECT * FROM udp_streams WHERE stream_id = ?`

	var stream models.UDPStream

	err := r.db.GetContext(ctx, &stream, query, streamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get udp stream with ID %d: %w", streamID.Int64, err)
	}
	return &stream, nil
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
	fmt.Println("Error : ", err)
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

/*New Func*/

func (r *Repository) GetPlaylistForDate(ctx context.Context, channelID int, PlaylistDate time.Time) (*models.Playlist, error) {
	//now := time.Now()
	//today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	fmt.Println("Playlist Date", PlaylistDate.Format("2006-01-02"))
	var playlist models.Playlist
	query := `SELECT * FROM playlists 
              WHERE channel_id = ? 
              AND (playlist_date = ? OR playlist_date IS NULL)
              
              ORDER BY playlist_date DESC
              LIMIT 1`

	err := r.db.GetContext(ctx, &playlist, query, channelID, PlaylistDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}

	return &playlist, nil
}

// GetNextPlaylistForChannel gets the next day's playlist
func (r *Repository) GetNextPlaylistForChannel(ctx context.Context, channelID int) (*models.Playlist, error) {
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

	var playlist models.Playlist
	query := `SELECT * FROM playlists 
              WHERE channel_id = ? 
              AND playlist_date = ?
              AND status = 'scheduled'
              LIMIT 1`

	err := r.db.GetContext(ctx, &playlist, query, channelID, tomorrow)
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

// GetCurrentAndNextPlaylistItems gets the current playing item and next queued item
func (r *Repository) GetCurrentAndNextPlaylistItems(ctx context.Context, playlistID int) (*models.PlaylistItem, *models.PlaylistItem, error) {
	now := time.Now()

	// Get current playing item (where scheduled_start_time <= now < scheduled_end_time)
	var currentItem models.PlaylistItem
	currentQuery := `SELECT * FROM playlist_items 
                    WHERE playlist_id = ? 
                    AND scheduled_start_time <= ? 
                    AND scheduled_end_time > ?
                    AND locked = true
                    LIMIT 1`

	err := r.db.GetContext(ctx, &currentItem, currentQuery, playlistID, now, now)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	// Get next item (first unlocked item after current item)
	var nextItem models.PlaylistItem
	nextQuery := `SELECT * FROM playlist_items 
                 WHERE playlist_id = ? 
                 AND position > ?
                 AND locked = false
                 ORDER BY position ASC
                 LIMIT 1`

	var position int
	if currentItem.ItemID > 0 {
		position = currentItem.Position
	} else {
		// If no current item, get first item
		position = -1
	}

	err = r.db.GetContext(ctx, &nextItem, nextQuery, playlistID, position)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	// If we're at the end of the playlist, loop to beginning
	if nextItem.ItemID == 0 {
		loopQuery := `SELECT * FROM playlist_items 
                     WHERE playlist_id = ? 
                     AND locked = false
                     ORDER BY position ASC
                     LIMIT 1`
		err = r.db.GetContext(ctx, &nextItem, loopQuery, playlistID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
	}

	return &currentItem, &nextItem, nil
}

// LockPlaylistItem marks an item as locked (currently playing or queued)
func (r *Repository) LockPlaylistItem(ctx context.Context, itemID int) error {
	query := `UPDATE playlist_items SET locked = true WHERE item_id = ?`
	_, err := r.db.ExecContext(ctx, query, itemID)
	return err
}

// UnlockPlaylistItem marks an item as unlocked
func (r *Repository) UnlockPlaylistItem(ctx context.Context, itemID int) error {
	query := `UPDATE playlist_items SET locked = false WHERE item_id = ?`
	_, err := r.db.ExecContext(ctx, query, itemID)
	return err
}
