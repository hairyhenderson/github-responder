package iptables

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/pkg/errors"
)

type ghMeta struct {
	Hooks    []*IPNet `json:"hooks,omitempty"`
	Git      []*IPNet `json:"git,omitempty"`
	Pages    []*IPNet `json:"pages,omitempty"`
	Importer []IPAddr `json:"importer,omitempty"`
}

func (m *ghMeta) String() string {
	s := "hooks: " + fmt.Sprintf("%s\n", m.Hooks)
	s += "git: " + fmt.Sprintf("%s\n", m.Git)
	s += "pages: " + fmt.Sprintf("%s\n", m.Pages)
	s += "importer: " + fmt.Sprintf("%s\n", m.Importer)
	return s
}

// IPNet - same as net.IPNet, but a json.Unmarshaler and json.Marshaler
type IPNet struct {
	*net.IPNet
}

// NewIPNet -
func NewIPNet(s string) (*IPNet, error) {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	return &IPNet{n}, nil
}

// UnmarshalJSON - fulfils json.Unmarshaler
func (i *IPNet) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	s := ""
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	n, err := NewIPNet(s)
	if err != nil {
		return err
	}
	*i = *n
	return nil
}

// MarshalJSON - fulfils json.Marshaler
func (i *IPNet) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// IPAddr - same as net.IPAddr, but a json.Unmarshaler and json.Marshaler
type IPAddr struct {
	net.IPAddr
}

// UnmarshalJSON - fulfils json.Unmarshaler
func (i *IPAddr) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}

	s := ""
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	ip := net.ParseIP(s)
	if ip == nil {
		return errors.Errorf("invalid IP format %v", ip)
	}
	*i = IPAddr{net.IPAddr{
		IP:   ip,
		Zone: "",
	}}

	return nil
}

// MarshalJSON - fulfils json.Marshaler
func (i IPAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}
