package main

import (
	stdlog "log"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

func initLogger() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	stdlogger := log.With().Bool("stdlog", true).Logger()

	stdlog.SetFlags(0)
	stdlog.SetOutput(stdlogger)

	if term.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})

		noLevelWriter := zerolog.ConsoleWriter{
			Out:         os.Stderr,
			FormatLevel: func(_ interface{}) string { return "" },
		}
		stdlogger = stdlogger.Output(noLevelWriter)
		stdlog.SetOutput(stdlogger)
	}
}
