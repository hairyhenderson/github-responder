package iptables

import (
	"strconv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/pkg/errors"
)

const (
	hookChainName = "github-hooks"
)

// GitHubHookRules -
type GitHubHookRules struct {
	ports []int
}

// New -
func New(ports ...int) (*GitHubHookRules, error) {
	for _, port := range ports {
		if port <= 0 {
			return nil, errors.Errorf("invalid port %d", port)
		}
	}
	return &GitHubHookRules{
		ports: ports,
	}, nil
}

// Init - sets up iptables rules so that GitHub's webhook source
// IP addresses can access the given ports. This creates a separate chain named
// `github-hooks`
func (g *GitHubHookRules) Init() error {
	meta, err := getGitHubMeta()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve GitHub metadata")
	}

	table := "filter"
	i, err := iptables.New()
	if err != nil {
		return err
	}
	err = i.ClearChain(table, hookChainName)
	if err != nil {
		return errors.Wrapf(err, "failed to clear chain %s", hookChainName)
	}
	for _, cidr := range meta.Hooks {
		i.Append(table, hookChainName, "-s", cidr.String(), "-j", "ACCEPT")
	}
	i.Append(table, hookChainName, "-j", "DROP")

	// now direct from specific ports
	for _, port := range g.ports {
		spec := []string{"-p", "tcp", "--dport", strconv.Itoa(port), "-j", hookChainName}
		exists, err := i.Exists(table, "INPUT", spec...)
		if err != nil {
			return errors.Wrapf(err, "failed to check existence of %s", spec)
		}
		if !exists {
			i.Append(table, "INPUT", spec...)
		}
	}
	return nil
}

// Cleanup - remove the rules that were created
func (g *GitHubHookRules) Cleanup() error {
	table := "filter"
	i, err := iptables.New()
	if err != nil {
		return err
	}
	err = i.ClearChain(table, hookChainName)
	if err != nil {
		return errors.Wrapf(err, "failed to clear chain %s", hookChainName)
	}
	for _, port := range g.ports {
		spec := []string{"-p", "tcp", "--dport", strconv.Itoa(port), "-j", hookChainName}
		exists, err := i.Exists(table, "INPUT", spec...)
		if err != nil {
			return errors.Wrapf(err, "failed to check existence of %s", spec)
		}
		if exists {
			i.Delete(table, "INPUT", spec...)
		}
	}
	return nil
}
