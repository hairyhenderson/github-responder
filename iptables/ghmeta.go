package iptables

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
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
		return nil, errors.Errorf("couldn't reach github meta endpoint %s: status %d (%s)", githubMeta, resp.StatusCode, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
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
