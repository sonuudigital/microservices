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
	logLevel := configLogLevel(os.Getenv("LOG_LEVEL"))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("logger initialized", "level", logLevel.String())

	return &SlogLogger{
		logger: logger,
	}
}

func configLogLevel(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
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
