package models

import "time"

type Overlay struct {
	OverlayID int       `json:"id" db:"id"`
	ChannelID int       `json:"channel_id" db:"channel_id"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	Type      string    `json:"type" db:"type"`
	FilePath  string    `json:"file_path" db:"file_path"`
	Text      string    `json:"text" db:"text"`
	PositionX string    `json:"position_x" db:"position_x"`
	PositionY string    `json:"position_y" db:"position_y"`
	FontSize  string    `json:"font_size" db:"font_size"`
	FontColor string    `json:"font_color" db:"font_color"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	FontFile  string    `json:"font_file" db:"-"`
}
