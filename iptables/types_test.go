package iptables

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPAddrString(t *testing.T) {
	addr := "127.0.0.1"
	i := IPAddr{net.IPAddr{IP: net.ParseIP(addr)}}
	assert.Equal(t, addr, i.String())
}

func TestIPAddrUnmarshalJSON(t *testing.T) {
	addr := "127.0.0.1"
	e := &IPAddr{net.IPAddr{IP: net.ParseIP(addr)}}
	a := &IPAddr{}
	err := json.Unmarshal([]byte(`"`+addr+`"`), a)
	assert.NoError(t, err)
	assert.EqualValues(t, e, a)
}

func TestIPAddrMarshalJSON(t *testing.T) {
	addr := "127.0.0.1"
	i := &IPAddr{net.IPAddr{IP: net.ParseIP(addr)}}
	b, err := json.Marshal(i)
	assert.NoError(t, err)
	assert.EqualValues(t, []byte(`"`+addr+`"`), b)
}

func TestIPNetString(t *testing.T) {
	addr := "127.0.0.1/24"
	_, n, _ := net.ParseCIDR(addr)
	i := &IPNet{n}
	assert.Equal(t, "127.0.0.0/24", i.String())
}

func TestIPNetUnmarshalJSON(t *testing.T) {
	addr := "127.0.0.1/22"
	_, n, _ := net.ParseCIDR(addr)
	e := &IPNet{n}
	a := &IPNet{}
	err := json.Unmarshal([]byte(`"`+addr+`"`), a)
	assert.NoError(t, err)
	assert.EqualValues(t, e, a)
}

func TestIPNetMarshalJSON(t *testing.T) {
	addr := "127.0.0.1/20"
	_, n, _ := net.ParseCIDR(addr)
	i := &IPNet{n}
	b, err := json.Marshal(i)
	assert.NoError(t, err)
	assert.EqualValues(t, []byte(`"127.0.0.0/20"`), b)
}

func TestNewIPNet(t *testing.T) {
	n, err := NewIPNet("127.0.0.0/24")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.0", n.IP.String())
	o, _ := n.Mask.Size()
	assert.Equal(t, 24, o)
}

func TestGhMeta(t *testing.T) {
	g := &ghMeta{
		Hooks: []*IPNet{
			must(NewIPNet("127.0.0.0/24")),
		},
	}

	a, err := json.Marshal(g)
	assert.NoError(t, err)
	assert.Equal(t, `{"hooks":["127.0.0.0/24"]}`, string(a))
}

func must(i *IPNet, err error) *IPNet {
	if err != nil {
		panic(err)
	}

	return i
}
