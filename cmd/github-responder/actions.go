package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	responder "github.com/hairyhenderson/github-responder"
)

func defaultAction(ctx context.Context, _, _ string, payload []byte) {
	slog.InfoContext(ctx, "Received event", "size", len(payload))

	j := make(map[string]interface{})

	err := json.Unmarshal(payload, &j)
	if err != nil {
		slog.ErrorContext(ctx, "Error parsing payload", "error", err)
	}

	pretty, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling payload", "error", err)
	}

	fmt.Println(string(pretty))
}

func execArgs(env []string, args ...string) responder.HookHandler {
	return func(ctx context.Context, eventType, deliveryID string, payload []byte) {
		name := args[0]
		cmdArgs := args[1:]
		cmdArgs = append(cmdArgs, eventType, deliveryID)
		input := bytes.NewBuffer(payload)

		c := exec.Command(name, cmdArgs...)
		c.Env = resolveEnv(env)
		slog.DebugContext(ctx, "Received event, executing command",
			"size", len(payload),
			"command", name,
			"args", cmdArgs,
			"env", keys(c.Env))

		c.Stdin = input
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout

		err := c.Run()
		if err != nil {
			slog.ErrorContext(ctx, err.Error(), "error", err)
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
