package responder

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
