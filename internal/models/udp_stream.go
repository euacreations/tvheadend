package models

import "time"

type UDPStream struct {
	StreamID        int       `json:"stream_id" db:"stream_id"`
	ChannelID       int       `json:"channel_id" db:"channel_id"`
	StreamName      string    `json:"stream_name" db:"stream_name"`
	StreamURL       string    `json:"stream_url" db:"stream_url"`
	Description     string    `json:"description" db:"description"`
	IsInfinite      bool      `json:"is_infinite" db:"is_infinite"`
	DurationSeconds *int      `json:"duration_seconds" db:"duration_seconds"` // Nil means infinite
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
