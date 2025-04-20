package models

import "time"

type MediaFile struct {
	ID              int
	ChannelID       int
	FilePath        string
	FileName        string
	DurationSeconds int
	ProgramName     string
	FileSize        int64
	LastModified    time.Time
	ScannedAt       time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
