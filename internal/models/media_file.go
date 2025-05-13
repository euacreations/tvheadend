package models

import "time"

type MediaFile struct {
	MediaID         int       `json:"media_id" db:"media_id"`
	ChannelID       int       `json:"channel_id" db:"channel_id"`
	FilePath        string    `json:"file_path" db:"file_path"`
	FileName        string    `json:"file_name" db:"file_name"`
	DurationSeconds int       `json:"duration_seconds" db:"duration_seconds"`
	ProgramName     string    `json:"program_name" db:"program_name"`
	FileSize        int64     `json:"file_size" db:"file_size"`
	LastModified    time.Time `json:"last_modified" db:"last_modified"`
	ScannedAt       time.Time `json:"scanned_at" db:"scanned_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
