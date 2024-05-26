package iptables

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	githubMeta = "https://api.github.com/meta"
)

func getGitHubMeta(ctx context.Context) (*ghMeta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubMeta, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	hc := http.DefaultClient

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("couldn't reach github meta endpoint %s: status %d (%s)", githubMeta, resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	r := &ghMeta{}

	err = json.Unmarshal(body, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
