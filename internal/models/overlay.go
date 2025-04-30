package models

import "time"

type Overlay struct {
	OverlayID int       `json:"overlay_id" db:"overlay_id"`
	ChannelID int       `json:"channel_id" db:"channel_id"`
	Type      string    `json:"type" db:"type"`
	FilePath  string    `json:"file_path" db:"file_path"`
	PositionX int       `json:"position_x" db:"position_x"`
	PositionY int       `json:"position_y" db:"position_y"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	FontSize  int       `json:"font_size" db:"font_size"`
	FontColor string    `json:"font_color" db:"font_color"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
