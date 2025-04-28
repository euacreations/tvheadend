package models

import "time"

type Playlist struct {
	ID                   int        `json:"id"`
	ChannelID            int        `json:"channel_id"`
	PlaylistDate         *time.Time `json:"playlist_date"` // NULL for infinite playlists
	Status               string     `json:"status"`        // scheduled, active, completed
	TotalDurationSeconds int        `json:"total_duration_seconds"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type PlaylistItem struct {
	ID                 int        `json:"id"`
	PlaylistID         int        `json:"playlist_id"`
	MediaID            int        `json:"media_id"`
	Position           int        `json:"position"`
	ScheduledStartTime *time.Time `json:"scheduled_start_time"`
	ScheduledEndTime   *time.Time `json:"scheduled_end_time"`
	ActualStartTime    *time.Time `json:"actual_start_time"`
	ActualEndTime      *time.Time `json:"actual_end_time"`
	Locked             bool       `json:"locked"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
