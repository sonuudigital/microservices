package logs

import (
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type SlogLogger struct {
	logger *slog.Logger
}

func NewSlogLogger() *SlogLogger {
	return &SlogLogger{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (s *SlogLogger) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *SlogLogger) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *SlogLogger) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *SlogLogger) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}
