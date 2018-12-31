package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeys(t *testing.T) {
	expected := []string{"foo", "bar", "baz"}
	pairs := []string{"foo=1", "bar=2", "baz"}
	assert.EqualValues(t, expected, keys(pairs))
}

func TestResolveEnv(t *testing.T) {
	expected := []string{"foo=1", "bar=2", "baz=", "USER=" + os.Getenv("USER")}
	pairs := []string{"foo=1", "bar=2", "baz", "USER"}
	assert.EqualValues(t, expected, resolveEnv(pairs))

	assert.EqualValues(t, os.Environ(), resolveEnv(nil))
	assert.EqualValues(t, os.Environ(), resolveEnv([]string{}))
}
