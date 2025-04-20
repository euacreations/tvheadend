package models

import "time"

type Overlay struct {
	ID        int
	ChannelID int
	Type      string // "image" or "text"
	FilePath  string // For image overlays
	PositionX int
	PositionY int
	Enabled   bool
	FontSize  int    // For text overlays
	FontColor string // For text overlays
	CreatedAt time.Time
	UpdatedAt time.Time
}
