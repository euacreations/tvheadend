package models

import "time"

type Channel struct {
	ID                     int
	Name                   string
	StorageRoot            string
	OutputUDP              string
	PlaylistType           string
	StartTime              time.Time
	Enabled                bool
	UsePreviousDayFallback bool
	State                  *ChannelState
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
