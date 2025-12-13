package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Logger zerolog.Logger

// Init initializes the global logger with appropriate configuration
func Init(logLevel string) {
	// Configure console output with colors and human-readable format for development
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	// Parse log level
	level := parseLogLevel(logLevel)

	// Create logger
	Logger = zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set as global logger
	log.Logger = Logger

	Logger.Info().
		Str("level", level.String()).
		Msg("Logger initialized")
}

// InitProduction initializes the logger for production with JSON output
func InitProduction(logLevel string, writers ...io.Writer) {
	// Default to stdout if no writers provided
	if len(writers) == 0 {
		writers = []io.Writer{os.Stdout}
	}

	output := io.MultiWriter(writers...)

	// Parse log level
	level := parseLogLevel(logLevel)

	// Create logger with JSON output for production
	Logger = zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set as global logger
	log.Logger = Logger

	Logger.Info().
		Str("level", level.String()).
		Msg("Production logger initialized")
}

func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// GetLogger returns the global logger instance
func GetLogger() *zerolog.Logger {
	return &Logger
}
