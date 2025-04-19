package models

import "time"

type ChannelState struct {
	ChannelID         int
	CurrentPlaylistID int
	CurrentItemID     int
	CurrentPosition   float64
	Running           bool
	FFmpegPID         int
	LastUpdateTime    time.Time
}
