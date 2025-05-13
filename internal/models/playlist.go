package models

import "time"

type Playlist struct {
	PlaylistID           int        `db:"playlist_id" json:"playlist_id"`
	ChannelID            int        `db:"channel_id" json:"channel_id"`
	PlaylistName         string     `db:"playlist_name" json:"playlist_name"`
	PlaylistDate         *time.Time `db:"playlist_date" json:"playlist_date"` // NULL for infinite playlists
	Status               string     `db:"status" json:"status"`               // scheduled, active, completed
	TotalDurationSeconds int        `db:"total_duration_seconds" json:"total_duration_seconds"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updated_at"`
}

type PlaylistItem struct {
	ItemID             int        `db:"item_id" json:"item_id"`
	PlaylistID         int        `db:"playlist_id" json:"playlist_id"`
	MediaID            int        `db:"media_id" json:"media_id"`
	Position           int        `db:"position" json:"position"`
	ScheduledStartTime *time.Time `db:"scheduled_start_time" json:"scheduled_start_time"`
	ScheduledEndTime   *time.Time `db:"scheduled_end_time" json:"scheduled_end_time"`
	ActualStartTime    *time.Time `db:"actual_start_time" json:"actual_start_time"`
	ActualEndTime      *time.Time `db:"actual_end_time" json:"actual_end_time"`
	Locked             bool       `db:"locked" json:"locked"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
}
