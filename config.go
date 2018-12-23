package responder

import (
	"github.com/rs/zerolog"
)

// Config - the github-responder configuration options
type Config struct {
	Owner  string
	Repo   string
	Events []string

	HookSecret  string
	GitHubToken string
	CallbackURL string

	// TLS options
	Domain      string
	Email       string
	HTTPPort    int
	HTTPSPort   int
	EnableTLS   bool
	StoragePath string
	CAEndpoint  string
}

var _ zerolog.LogObjectMarshaler = Config{}

// MarshalZerologObject - satisfies zerolog.LogObjectMarshaler
func (c Config) MarshalZerologObject(e *zerolog.Event) {
	e.Str("owner", c.Owner).
		Str("repo", c.Repo).
		Strs("events", c.Events).
		Str("callback_url", c.CallbackURL).
		Str("domain", c.Domain).
		Str("email", c.Email).
		Int("http_port", c.HTTPPort).
		Int("https_port", c.HTTPSPort).
		Bool("enable_tls", c.EnableTLS).
		Str("path", c.StoragePath).
		Str("ca", c.CAEndpoint)
}
