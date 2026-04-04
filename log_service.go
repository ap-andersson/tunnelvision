package main

import (
	"context"

	logstreamer "github.com/ap-andersson/tunnelvision/internal/log"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LogService exposes live journal logs to the frontend.
type LogService struct {
	ctx      context.Context
	streamer *logstreamer.Streamer
}

// NewLogService creates a new LogService.
func NewLogService(streamer *logstreamer.Streamer) *LogService {
	return &LogService{streamer: streamer}
}

// SetContext stores the Wails runtime context (called from OnStartup).
func (l *LogService) SetContext(ctx context.Context) {
	l.ctx = ctx
	l.streamer.SetListener(func(line string) {
		if l.ctx != nil {
			runtime.EventsEmit(l.ctx, "log:line", line)
		}
	})
}

// GetLogs returns the buffered log lines.
func (l *LogService) GetLogs() []string {
	lines := l.streamer.Lines()
	if lines == nil {
		return []string{}
	}
	return lines
}
