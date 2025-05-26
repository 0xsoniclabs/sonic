package logger

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

// init with defaults.
func init() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
}

// SetLevel sets level filter on log root handler.
// So it should be called last.
func SetLevel(l string) {
	lvl, err := levelFromString(l)
	if err != nil {
		panic(err)
	}

	log.SetDefault(
		log.NewLogger(
			log.NewTerminalHandlerWithLevel(os.Stderr, lvl, true),
		))
}

func levelFromString(lvlString string) (slog.Level, error) {
	switch lvlString {
	case "debug", "dbug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error", "eror":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown level: %v", lvlString)
	}
}

//go:generate mockgen -source=logger.go -destination=logger_mock.go -package=logger

// Logger defined as an alias for log.Logger to allow mocking in tests.
type Logger interface {
	log.Logger
}
