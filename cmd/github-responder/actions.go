package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	responder "github.com/hairyhenderson/github-responder"
	"github.com/rs/zerolog/log"
)

func defaultAction(ctx context.Context, eventType, deliveryID string, payload []byte) {
	log := log.Ctx(ctx)
	log.Info().
		Int("size", len(payload)).
		Msg("Received event")

	j := make(map[string]interface{})
	err := json.Unmarshal(payload, &j)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing payload")
	}

	pretty, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		log.Error().Err(err).Msg("Error unmarshaling payload")
	}
	fmt.Println(string(pretty))
}

func execArgs(args ...string) responder.HookHandler {
	return func(ctx context.Context, eventType, deliveryID string, payload []byte) {
		log := log.Ctx(ctx)
		name := args[0]
		cmdArgs := args[1:]
		cmdArgs = append(cmdArgs, eventType, deliveryID)
		log.Debug().
			Int("size", len(payload)).
			Str("command", name).
			Strs("args", cmdArgs).
			Msg("Received event, executing command")
		input := bytes.NewBuffer(payload)
		// nolint: gosec
		c := exec.Command(name, cmdArgs...)
		c.Stdin = input
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		err := c.Run()
		if err != nil {
			log.Error().Err(err).Msg(err.Error())
		}
	}
}
