package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/euacreations/tvheadend/internal/models"
)

type StreamConfig struct {
	// Core Parameters
	InputPath   string
	OutputURL   string
	StartOffset time.Duration
	Duration    time.Duration
	// Video Parameters
	VideoCodec       string
	VideoBitrate     string
	MinBitrate       string
	MaxBitrate       string
	BufferSize       string
	OutputResolution string

	// Audio Parameters
	AudioCodec   string
	AudioBitrate string

	// MPEG-TS Metadata
	PacketSize              int
	MpegTSOriginalNetworkID int
	MpegTSTransportStreamID int
	MpegTSServiceID         int
	MpegTSStartPID          int
	MpegTSPMTStartPID       int
	MetadataServiceProvider string
	MmetadataServiceName    string

	Overlays []models.Overlay
}

const (
	OverlayTypeText  string = "text"
	OverlayTypeImage string = "image"
)

type Streamer struct {
	cmd             *exec.Cmd
	running         bool
	pid             int
	mux             sync.Mutex
	logBuffer       strings.Builder
	currentPosition float64
	done            chan struct{}
	onProgress      func(position float64)
}

func New() *Streamer {
	return &Streamer{
		done: make(chan struct{}),
	}

}

func (s *Streamer) Start(ctx context.Context, config StreamConfig) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.running {
		return fmt.Errorf("stream is already running")
	}

	args := []string{
		"-init_hw_device", "cuda=cu:0",
		"-filter_hw_device", "cu",
		"-hwaccel", "cuda",
		"-hwaccel_output_format", "cuda",
		"-re",
	}

	// Add start offset if specified
	if config.StartOffset > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.2f", config.StartOffset.Seconds()))
	}

	// Add duration if specified
	if config.Duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.2f", config.Duration.Seconds()))
	}

	// Input specification
	args = append(args, "-i", config.InputPath)

	// Add additional overlay image inputs
	for _, overlay := range config.Overlays {
		if overlay.Type == OverlayTypeImage {
			args = append(args, "-i", overlay.FilePath) // Add image overlay input
		}
	}

	// Video encoding parameters
	args = append(args,
		"-c:v", config.VideoCodec,
		"-b:v", config.VideoBitrate,
		"-minrate", config.MinBitrate,
		"-maxrate", config.MaxBitrate,
		"-bufsize", config.BufferSize,
	)
	// Audio encoding parameters
	args = append(args,
		"-c:a", config.AudioCodec,
		"-b:a", config.AudioBitrate,
	)

	if len(config.Overlays) > 0 {
		args = append(args, "-filter_complex", s.buildOverlayFilter(config.Overlays, config.OutputResolution))
	}
	args = append(args, "-map", "[outv]", "-map", "0:a")

	// MPEG-TS parameters
	args = append(args,
		"-pkt_size", strconv.Itoa(config.PacketSize),
		"-mpegts_original_network_id", fmt.Sprintf("%d", config.MpegTSOriginalNetworkID),
		"-mpegts_transport_stream_id", fmt.Sprintf("%d", config.MpegTSTransportStreamID),
		"-mpegts_service_id", strconv.Itoa(config.MpegTSServiceID),
		"-mpegts_start_pid", strconv.Itoa(config.MpegTSStartPID),
		"-mpegts_pmt_start_pid", strconv.Itoa(config.MpegTSPMTStartPID),
		"-metadata", fmt.Sprintf("service_provider='%s'", config.MetadataServiceProvider),
		"-metadata", fmt.Sprintf("service_name='%s'", config.MmetadataServiceName),
	)

	// Output format
	args = append(args, "-f", "mpegts", config.OutputURL)

	args = append(args, "-progress", "pipe:2")

	//fmt.Println("FFmpeg command:", args)

	s.cmd = exec.CommandContext(context.Background(), "ffmpeg", args...)

	// Setup stderr log capture
	stderrPipe, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get FFmpeg stderr: %w", err)
	}
	s.currentPosition = 0
	//s.logBuffer.Reset()
	//s.startLogParser(stderrPipe)

	go func() {
		buf := make([]byte, 1024)
		lineBuf := ""

		for {
			n, err := stderrPipe.Read(buf)

			if n > 0 {
				lineBuf += string(buf[:n])

				for {
					idx := strings.Index(lineBuf, "\n")
					if idx == -1 {
						break
					}
					line := strings.TrimSpace(lineBuf[:idx])
					lineBuf = lineBuf[idx+1:]

					// Parse FFmpeg progress key=value lines
					if strings.HasPrefix(line, "out_time=") {
						timestamp := strings.TrimPrefix(line, "out_time=")
						position, err := parseFFmpegTime(timestamp)
						if err == nil {
							s.mux.Lock()
							s.currentPosition = position + config.StartOffset.Seconds() // <-- float64 in seconds
							s.mux.Unlock()
						}

					}
				}
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.mux.Lock()
				position := s.currentPosition
				s.mux.Unlock()

				// Call back to update DB
				if s.onProgress != nil {
					s.onProgress(position)
				}
			case <-s.done:
				return
			}
		}
	}()

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	s.running = true
	s.pid = s.cmd.Process.Pid

	s.done = make(chan struct{})

	go func() {
		_ = s.cmd.Wait()
		s.mux.Lock()
		s.running = false
		s.mux.Unlock()
		close(s.done)
	}()

	return nil
}

func (s *Streamer) Reset() {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.running {
		close(s.done) // Close the existing done channel
		s.cmd.Process.Signal(syscall.SIGTERM)
	}
	s.running = false
	s.currentPosition = 0
	s.logBuffer.Reset()
	s.cmd = nil
	s.pid = 0
	s.onProgress = nil           // Clear the progress callback
	s.done = make(chan struct{}) // Create a new done channel
}

func parseFFmpegTime(timeStr string) (float64, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format")
	}
	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	secs, _ := strconv.ParseFloat(parts[2], 64)
	return float64(hours*3600+minutes*60) + secs, nil
}

func (s *Streamer) SetProgressCallback(callback func(position float64)) {
	s.onProgress = callback
}

func (s *Streamer) buildOverlayFilter(overlays []models.Overlay, outputResolution string) string {
	var filters []string
	currentLabel := "0:v"
	filterIndex := 0
	imageCount := 1 // FFmpeg input indices start from 1 for overlays

	// Scale input video
	scaledLabel := fmt.Sprintf("v%d", filterIndex)
	filters = append(filters, fmt.Sprintf("[%s]scale_cuda=%s[%s]", currentLabel, outputResolution, scaledLabel))
	currentLabel = scaledLabel
	filterIndex++

	// hwdownload and format=nv12
	downloadLabel := fmt.Sprintf("v%d", filterIndex)
	filters = append(filters, fmt.Sprintf("[%s]hwdownload,format=nv12[%s]", currentLabel, downloadLabel))
	currentLabel = downloadLabel
	filterIndex++

	// Apply overlays
	for _, overlay := range overlays {
		switch overlay.Type {
		case OverlayTypeText:
			textLabel := fmt.Sprintf("v%d", filterIndex)
			filters = append(filters, fmt.Sprintf(
				"[%s]drawtext=fontfile='%s':text='%s':x=%s:y=%s:fontsize=%s:fontcolor=%s:shadowcolor=black:shadowx=2:shadowy=2[%s]",
				currentLabel,
				overlay.FontFile,
				overlay.Text,
				overlay.PositionX,
				overlay.PositionY,
				overlay.FontSize,
				overlay.FontColor,
				textLabel,
			))
			currentLabel = textLabel
			filterIndex++
		case OverlayTypeImage:
			overlayLabel := fmt.Sprintf("v%d", filterIndex)
			filters = append(filters, fmt.Sprintf(
				"[%s][%d:v]overlay=%s:%s[%s]",
				currentLabel,
				imageCount,
				overlay.PositionX,
				overlay.PositionY,
				overlayLabel,
			))
			currentLabel = overlayLabel
			filterIndex++
			imageCount++
		}
	}

	filters = append(filters, fmt.Sprintf("[%s]format=nv12,hwupload_cuda,format=cuda[outv]", currentLabel))

	return strings.Join(filters, ";")
}

func (s *Streamer) Stop() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if !s.running || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop FFmpeg: %w", err)
	}

	s.running = false
	return nil
}

func (s *Streamer) IsRunning() bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	if !s.running || s.pid == 0 {
		return false
	}

	// Linux: Check if the process directory exists
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", s.pid)); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// For other unexpected errors, assume not running
		return false
	}

	return true
}

func (s *Streamer) PID() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.pid
}

func (s *Streamer) Done() <-chan struct{} {
	return s.done
}
