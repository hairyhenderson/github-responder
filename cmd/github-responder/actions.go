package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/dustin/go-humanize"
)

func defaultAction(eventType, deliveryID string, payload []byte) {
	log.Printf("Received %s event %s with payload of %s", eventType, deliveryID, humanize.Bytes(uint64(len(payload))))
	j := make(map[string]interface{})
	err := json.Unmarshal(payload, &j)
	if err != nil {
		log.Printf("Error parsing payload: %v", err)
	}

	pretty, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		log.Printf("Error unmarshaling payload: %v", err)
	}
	fmt.Println(string(pretty))
}

func execArgs(args ...string) func(eventType, deliveryID string, payload []byte) {
	return func(eventType, deliveryID string, payload []byte) {
		name := args[0]
		cmdArgs := args[1:]
		cmdArgs = append(cmdArgs, eventType, deliveryID)
		logf("exec.Command(%s, %v) with input of %db", name, cmdArgs, len(payload))
		input := bytes.NewBuffer(payload)
		// nolint: gosec
		c := exec.Command(name, cmdArgs...)
		c.Stdin = input
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		err := c.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
}
