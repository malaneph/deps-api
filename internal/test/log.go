package test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
)

type LogSpy struct {
	mu      sync.Mutex
	records []slog.Record
}

func (s *LogSpy) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (s *LogSpy) Handle(_ context.Context, r slog.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, r)
	return nil
}

func (s *LogSpy) WithAttrs(_ []slog.Attr) slog.Handler { return s }
func (s *LogSpy) WithGroup(_ string) slog.Handler      { return s }

func (s *LogSpy) Records() []slog.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]slog.Record, len(s.records))
	copy(out, s.records)
	return out
}

func (s *LogSpy) HasLevel(level slog.Level) bool {
	for _, r := range s.Records() {
		if r.Level == level {
			return true
		}
	}
	return false
}

func (s *LogSpy) HasMessage(msg string) bool {
	for _, r := range s.Records() {
		if r.Message == msg {
			return true
		}
	}
	return false
}

// NewLogSpy replaces the global slog logger for the duration of t and returns
// a spy that collects all records emitted during that time.
func NewLogSpy(t *testing.T) *LogSpy {
	t.Helper()
	spy := &LogSpy{}
	prev := slog.Default()
	slog.SetDefault(slog.New(spy))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return spy
}
