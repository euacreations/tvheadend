package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
)

type StreamConfig struct {
	InputPath    string
	OutputURL    string
	VideoCodec   string
	VideoBitrate string
	MinBitrate   string
	MaxBitrate   string
	BufferSize   string
}

type Streamer struct {
	cmd     *exec.Cmd
	running bool
	pid     int
	mux     sync.Mutex
}

func New() *Streamer {
	return &Streamer{}
}

func (s *Streamer) Start(ctx context.Context, config StreamConfig) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.running {
		return fmt.Errorf("stream is already running")
	}

	args := []string{
		"-hwaccel", "cuda",
		"-i", config.InputPath,
		"-c:v", config.VideoCodec,
		"-b:v", config.VideoBitrate,
		"-minrate", config.MinBitrate,
		"-maxrate", config.MaxBitrate,
		"-bufsize", config.BufferSize,
		"-f", "mpegts",
		config.OutputURL,
	}

	s.cmd = exec.CommandContext(ctx, "ffmpeg", args...)
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	s.running = true
	s.pid = s.cmd.Process.Pid

	go func() {
		_ = s.cmd.Wait()
		s.mux.Lock()
		s.running = false
		s.mux.Unlock()
	}()

	return nil
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
	return s.running
}

func (s *Streamer) PID() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.pid
}
