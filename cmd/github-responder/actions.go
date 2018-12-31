package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

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

func execArgs(env []string, args ...string) responder.HookHandler {
	return func(ctx context.Context, eventType, deliveryID string, payload []byte) {
		log := log.Ctx(ctx)
		name := args[0]
		cmdArgs := args[1:]
		cmdArgs = append(cmdArgs, eventType, deliveryID)
		input := bytes.NewBuffer(payload)
		// nolint: gosec
		c := exec.Command(name, cmdArgs...)
		c.Env = resolveEnv(env)
		log.Debug().
			Int("size", len(payload)).
			Str("command", name).
			Strs("args", cmdArgs).
			Strs("env", keys(c.Env)).
			Msg("Received event, executing command")
		c.Stdin = input
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		err := c.Run()
		if err != nil {
			log.Error().Err(err).Msg(err.Error())
		}
	}
}

func keys(kvPairs []string) []string {
	out := make([]string, len(kvPairs))
	for i, kv := range kvPairs {
		parts := strings.SplitN(kv, "=", 2)
		out[i] = parts[0]
	}
	return out
}

func resolveEnv(kvPairs []string) []string {
	if len(kvPairs) == 0 {
		return os.Environ()
	}

	out := make([]string, len(kvPairs))
	for i, kv := range kvPairs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 1 {
			out[i] = parts[0] + "=" + os.Getenv(parts[0])
		} else {
			out[i] = kv
		}
	}
	return out
}
