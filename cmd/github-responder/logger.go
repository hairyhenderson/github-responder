package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh/terminal"
)

func initLogger() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
	}
}

func setVerboseLogging() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}
