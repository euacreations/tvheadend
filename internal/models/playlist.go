package models

import "time"

type Playlist struct {
	ID                   int
	ChannelID            int
	PlaylistDate         *time.Time // NULL for infinite playlists
	Status               string     // scheduled, active, completed
	TotalDurationSeconds int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type PlaylistItem struct {
	ID                 int
	PlaylistID         int
	MediaID            int
	Position           int
	ScheduledStartTime *time.Time
	ScheduledEndTime   *time.Time
	ActualStartTime    *time.Time
	ActualEndTime      *time.Time
	Locked             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
