package models

import "time"

type Channel struct {
	ChannelID               int           `json:"channel_id" db:"channel_id"`
	ChannelName             string        `json:"channel_name" db:"channel_name"`
	StorageRoot             string        `json:"storage_root" db:"storage_root"`
	OutputUDP               string        `json:"output_udp" db:"output_udp"`
	PlaylistType            string        `json:"playlist_type" db:"playlist_type"`
	PlaylistID              int           `json:"playlist_id" db:"playlist_id"`
	StartTimeStr            string        `json:"-" db:"start_time" `
	StartTime               time.Time     `json:"start_time" db:"-" `
	Enabled                 bool          `json:"enabled" db:"enabled"`
	UsePreviousDayFallback  bool          `json:"use_previous_day_fallback" db:"use_previous_day_fallback"`
	VideoCodec              string        `json:"video_codec" db:"video_codec"`
	VideoBitrate            string        `json:"video_bitrate" db:"video_bitrate"`
	MinBitrate              string        `json:"min_bitrate" db:"min_bitrate"`
	MaxBitrate              string        `json:"max_bitrate" db:"max_bitrate"`
	AudioCodec              string        `json:"audio_codec" db:"audio_codec"`
	AudioBitrate            string        `json:"audio_bitrate" db:"audio_bitrate"`
	BufferSize              string        `json:"buffer_size" db:"buffer_size"`
	PacketSize              int           `json:"packet_size" db:"packet_size"`
	OutputResolution        string        `json:"output_resolution" db:"output_resolution"`
	MPEGTSOriginalNetworkID int           `json:"mpegts_original_network_id" db:"mpegts_original_network_id"`
	MPEGTSTransportStreamID int           `json:"mpegts_transport_stream_id" db:"mpegts_transport_stream_id"`
	MPEGTSServiceID         int           `json:"mpegts_service_id" db:"mpegts_service_id"`
	MPEGTSStartPID          int           `json:"mpegts_start_pid" db:"mpegts_start_pid"`
	MPEGTSPMTStartPID       int           `json:"mpegts_pmt_start_pid" db:"mpegts_pmt_start_pid"`
	MetadataServiceProvider string        `json:"metadata_service_provider" db:"metadata_service_provider"`
	State                   *ChannelState `json:"state" db:"-"`
	CreatedAt               time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at" db:"updated_at"`
}
