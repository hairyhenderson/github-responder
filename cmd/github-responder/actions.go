package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

func defaultAction(eventType, deliveryID string, payload []byte) {
	log.Info().
		Str("eventType", eventType).
		Str("deliveryID", deliveryID).
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

func execArgs(args ...string) func(eventType, deliveryID string, payload []byte) {
	return func(eventType, deliveryID string, payload []byte) {
		name := args[0]
		cmdArgs := args[1:]
		cmdArgs = append(cmdArgs, eventType, deliveryID)
		log.Debug().
			Str("eventType", eventType).
			Str("deliveryID", deliveryID).
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
