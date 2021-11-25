package iptables

import (
	"fmt"
	"strconv"

	"github.com/coreos/go-iptables/iptables"
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
			return nil, fmt.Errorf("invalid port %d", port)
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
		return fmt.Errorf("failed to retrieve GitHub metadata: %w", err)
	}

	table := "filter"
	i, err := iptables.New()
	if err != nil {
		return err
	}
	err = i.ClearChain(table, hookChainName)
	if err != nil {
		return fmt.Errorf("failed to clear chain %s: %w", hookChainName, err)
	}
	for _, cidr := range meta.Hooks {
		err = i.Append(table, hookChainName, "-s", cidr.String(), "-j", "ACCEPT")
		if err != nil {
			return fmt.Errorf("failed to append rule for %s: %w", cidr, err)
		}
	}
	err = i.Append(table, hookChainName, "-j", "DROP")
	if err != nil {
		return fmt.Errorf("failed to append default rule for %s: %w", hookChainName, err)
	}

	// now direct from specific ports
	for _, port := range g.ports {
		err = appendPortRule(port, i, table, hookChainName)
		if err != nil {
			return err
		}
	}
	return nil
}

func appendPortRule(port int, i *iptables.IPTables, table, chain string) error {
	spec := []string{"-p", "tcp", "--dport", strconv.Itoa(port), "-j", chain}
	exists, err := i.Exists(table, "INPUT", spec...)
	if err != nil {
		return fmt.Errorf("failed to check existence of %s: %w", spec, err)
	}
	if !exists {
		err = i.Append(table, "INPUT", spec...)
		if err != nil {
			return fmt.Errorf("failed to append rule for port %d: %w", port, err)
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
		return fmt.Errorf("failed to clear chain %s: %w", hookChainName, err)
	}
	for _, port := range g.ports {
		spec := []string{"-p", "tcp", "--dport", strconv.Itoa(port), "-j", hookChainName}
		exists, err := i.Exists(table, "INPUT", spec...)
		if err != nil {
			return fmt.Errorf("failed to check existence of %s: %w", spec, err)
		}
		if exists {
			err = i.Delete(table, "INPUT", spec...)
			if err != nil {
				return fmt.Errorf("failed to delete rule for %s: %w", hookChainName, err)
			}
		}
	}
	return nil
}
