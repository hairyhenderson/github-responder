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
	Accept      bool
	HTTPAddress string
	TLSAddress  string
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
		Bool("accept", c.Accept).
		Str("http_addr", c.HTTPAddress).
		Str("tls_addr", c.TLSAddress).
		Bool("enable_tls", c.EnableTLS).
		Str("path", c.StoragePath).
		Str("ca", c.CAEndpoint)
}
