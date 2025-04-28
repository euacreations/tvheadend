package models

import "time"

type Overlay struct {
	ID        int       `json:"id"`
	ChannelID int       `json:"channel_id"`
	Type      string    `json:"type"`      // "image" or "text"
	FilePath  string    `json:"file_path"` // For image overlays
	PositionX int       `json:"position_x"`
	PositionY int       `json:"position_y"`
	Enabled   bool      `json:"enabled"`
	FontSize  int       `json:"font_size"`  // For text overlays
	FontColor string    `json:"font_color"` // For text overlays
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
