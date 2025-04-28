package models

import "time"

type MediaFile struct {
	ID              int       `json:"id"`
	ChannelID       int       `json:"channel_id"`
	FilePath        string    `json:"file_path"`
	FileName        string    `json:"file_name"`
	DurationSeconds int       `json:"duration_seconds"`
	ProgramName     string    `json:"program_name"`
	FileSize        int64     `json:"file_size"`
	LastModified    time.Time `json:"last_modified"`
	ScannedAt       time.Time `json:"scanned_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
