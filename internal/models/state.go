package models

import "time"

type ChannelState struct {
	StateID           int       `json:"state_id" db:"state_id"`
	ChannelID         int       `json:"channel_id" db:"channel_id"`
	CurrentPlaylistID int       `json:"current_playlist_id" db:"current_playlist_id"`
	CurrentItemID     int       `json:"current_item_id" db:"current_item_id"`
	CurrentPosition   float64   `json:"current_position" db:"current_position"`
	Running           bool      `json:"running" db:"running"`
	FFmpegPID         int       `json:"ffmpeg_pid" db:"ffmpeg_pid"`
	LastUpdateTime    time.Time `json:"last_update_time" db:"last_update_time"`
	ErrorMessage      *string   `json:"error_message" db:"error_message"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}
