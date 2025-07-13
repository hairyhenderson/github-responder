package main

import (
	stdlog "log"
	"log/slog"
	"os"

	"golang.org/x/term"
)

func initLogger() {
	// Set global log level to Info
	level := slog.LevelInfo

	var handler slog.Handler

	if term.IsTerminal(int(os.Stdout.Fd())) {
		// Terminal mode: use text handler with custom formatting
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				// Format time as HH:MM:SS
				if a.Key == slog.TimeKey {
					return slog.String("time", a.Value.Time().Format("15:04:05"))
				}

				return a
			},
		})
	} else {
		// Non-terminal mode: use JSON handler
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	}

	// Set the default logger
	slog.SetDefault(slog.New(handler))

	// Redirect standard log to slog
	stdlog.SetFlags(0)
	stdlog.SetOutput(&slogWriter{})
}

// slogWriter adapts slog to io.Writer for standard log redirection
type slogWriter struct{}

func (w *slogWriter) Write(p []byte) (n int, err error) {
	// Remove trailing newline if present
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	// Log with stdlog=true attribute to distinguish from direct slog calls
	slog.Info(msg, "stdlog", true)

	return len(p), nil
}
