package iptables

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	githubMeta = "https://api.github.com/meta"
)

func getGitHubMeta() (*ghMeta, error) {
	resp, err := http.Get(githubMeta)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
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
