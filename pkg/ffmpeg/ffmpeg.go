package ffmpeg

import (
	"context"
	"fmt"
	"io"
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
	InputType   models.PlaylistItemType
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
	stopOnce        sync.Once // Ensures cleanup happens only once
	ctx             context.Context
	cancel          context.CancelFunc
}

func New() *Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Streamer{
		done:   make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
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
	}

	// Use -f mpegts and -async for UDP input format
	if config.InputType == models.PlaylistItemTypeUDP {
		args = append(args, "-f", "mpegts")
		args = append(args, "-async", "30")

	} else if config.InputType == models.PlaylistItemTypeMedia {
		args = append(args, "-re") // Use -re for file input to simulate real-time

	}

	// Place -ss and -t before -i for faster input-level seek/truncate
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

	args = append(args, "-preset", "p1") // p1 is fastest (ultrafast), p7 is slowest/best quality
	args = append(args, "-tune", "ull")  // Tune for ultra-low latency
	args = append(args, "-rc", "cbr")    // Use constant bitrate for predictable bandwidth

	args = append(args, "-g", "60")           // GOP size (2 sec at 30 fps)
	args = append(args, "-keyint_min", "60")  // Minimum GOP size
	args = append(args, "-sc_threshold", "0") // Disable scene-change triggers for keyframes

	args = append(args, "-r", "30") // Output frame rate

	// Audio encoding parameters
	args = append(args,
		"-c:a", config.AudioCodec,
		"-b:a", config.AudioBitrate,
	)

	if len(config.Overlays) > 0 {
		args = append(args, "-filter_complex", s.buildOverlayFilter(config.Overlays, config.OutputResolution))
		args = append(args, "-map", "[outv]", "-map", "0:a")
	}

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

	// Use the streamer's context for the command
	s.cmd = exec.CommandContext(s.ctx, "ffmpeg", args...)

	// Setup stderr log capture
	stderrPipe, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get FFmpeg stderr: %w", err)
	}
	s.currentPosition = 0

	// Progress parsing goroutine
	go s.parseProgress(stderrPipe, config.StartOffset)

	// Progress callback goroutine
	go s.progressCallback()

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	s.running = true
	s.pid = s.cmd.Process.Pid

	// Process monitoring goroutine
	go s.monitorProcess()

	return nil
}

func (s *Streamer) parseProgress(stderrPipe io.ReadCloser, startOffset time.Duration) {
	defer stderrPipe.Close()

	buf := make([]byte, 1024)
	lineBuf := ""

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

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
						s.currentPosition = position + startOffset.Seconds()
						s.mux.Unlock()
					}
				}
			}
		}
		if err != nil {
			break
		}
	}
}

func (s *Streamer) progressCallback() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mux.Lock()
			position := s.currentPosition
			callback := s.onProgress
			s.mux.Unlock()

			// Call back to update DB
			if callback != nil {
				callback(position)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Streamer) monitorProcess() {
	_ = s.cmd.Wait()

	// Use sync.Once to ensure cleanup happens only once
	s.stopOnce.Do(func() {
		s.mux.Lock()
		s.running = false
		s.mux.Unlock()

		// Close the done channel to signal completion
		select {
		case <-s.done:
			// Channel already closed
		default:
			close(s.done)
		}
	})
}

func (s *Streamer) Reset() {
	// Cancel the context to stop all goroutines
	s.cancel()

	s.mux.Lock()
	defer s.mux.Unlock()

	// Kill the process if it's running
	if s.running && s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Signal(syscall.SIGTERM)

		// Give it a moment to terminate gracefully
		time.Sleep(100 * time.Millisecond)

		// Force kill if still running
		if s.IsRunning() {
			s.cmd.Process.Kill()
		}
	}

	// Reset state
	s.running = false
	s.currentPosition = 0
	s.logBuffer.Reset()
	s.cmd = nil
	s.pid = 0
	s.onProgress = nil

	// Ensure done channel is closed
	s.stopOnce.Do(func() {
		select {
		case <-s.done:
			// Channel already closed
		default:
			close(s.done)
		}
	})

	// Create new context and channels for potential reuse
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.done = make(chan struct{})
	s.stopOnce = sync.Once{}
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
	s.mux.Lock()
	defer s.mux.Unlock()
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
