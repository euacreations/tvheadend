package models

import "time"

type Channel struct {
	ID                     int           `json:"id"`
	Name                   string        `json:"name"`
	StorageRoot            string        `json:"storage_root"`
	OutputUDP              string        `json:"output_udp"`
	PlaylistType           string        `json:"playlist_type"`
	StartTime              time.Time     `json:"start_time"`
	Enabled                bool          `json:"enabled"`
	UsePreviousDayFallback bool          `json:"use_previous_day_fallback"`
	State                  *ChannelState `json:"state"`
	CreatedAt              time.Time     `json:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at"`
}
