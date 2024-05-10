package logger

import (
	"log/slog"
	"os"
)

var (
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	Debug  = Logger.Debug
	Info   = Logger.Info
	Warn   = Logger.Warn
	Error  = Logger.Error
)
