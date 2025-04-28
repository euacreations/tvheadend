package models

import "time"

type ChannelState struct {
	ChannelID         int       `json:"channel_id"`
	CurrentPlaylistID int       `json:"current_playlist_id"`
	CurrentItemID     int       `json:"current_item_id"`
	CurrentPosition   float64   `json:"current_position"`
	Running           bool      `json:"running"`
	FFmpegPID         int       `json:"ffmpeg_pid"`
	LastUpdateTime    time.Time `json:"last_update_time"`
}
