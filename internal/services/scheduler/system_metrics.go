package scheduler

import (
	"context"
	"log/slog"
	"time"
)

type metricsRecorder interface {
	RecordSnapshot(ctx context.Context) error
}

type SystemMetricsCollector struct {
	recorder metricsRecorder
	logger   *slog.Logger
}

func NewSystemMetricsCollector(recorder metricsRecorder, logger *slog.Logger) *SystemMetricsCollector {
	return &SystemMetricsCollector{recorder: recorder, logger: logger}
}

func (c *SystemMetricsCollector) Run(ctx context.Context) {
	c.collect(ctx)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

func (c *SystemMetricsCollector) collect(ctx context.Context) {
	if c.recorder == nil {
		return
	}
	if err := c.recorder.RecordSnapshot(ctx); err != nil {
		c.logger.Warn("system metrics snapshot failed", "error", err)
	}
}
