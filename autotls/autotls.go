package autotls

import (
	"context"
	"log"
	"net/url"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/xenolf/lego/acme"
)

const (
	// LetsEncryptProductionURL - the production Let's Encrypt CA URL
	LetsEncryptProductionURL = "https://acme-v02.api.letsencrypt.org/directory"

	// LetsEncryptStagingURL - the staging Let's Encrypt CA URL - use this for testing
	LetsEncryptStagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

// AutoTLS provides the ability to automatically retrieve TLS certificates from
// Let's Encyrpt with a minimum of configuration.
type AutoTLS struct {
	Accept bool
	// Domain for which we want a cert
	Domain string
	// Email address to register the user account with
	Email string
	// address (ip:port) to use for HTTP-based challenges
	HTTPAddress string
	// address (ip:port) to use for TLS-based challenges
	TLSAddress string
	// location to store user, certs, and keys - use ~ for home dir
	StoragePath string
	// the ACME directory URL - can be overridden to try with test CAs
	CAEndpoint string
}

// New -
func New(domain, email string) *AutoTLS {
	defaultStoragePath := filepath.Join("~", ".lego")
	return &AutoTLS{
		Domain:      domain,
		Email:       email,
		StoragePath: defaultStoragePath,
		HTTPAddress: ":80",
		TLSAddress:  ":443",
		CAEndpoint:  LetsEncryptProductionURL,
	}
}

// getStoragePath -
func (t *AutoTLS) getStoragePath() string {
	s, err := homedir.Expand(t.StoragePath)
	if err != nil {
		s = t.StoragePath
	}
	return s
}

// CertPaths returns the path to the certificate and the private key
func (t *AutoTLS) CertPaths() (string, string) {
	ep := getHost(t.CAEndpoint)
	certPath := filepath.Join(t.getStoragePath(), ep, "sites", t.Domain)
	certFile := filepath.Join(certPath, t.Domain+".crt")
	keyFile := filepath.Join(certPath, t.Domain+".key")
	return certFile, keyFile
}

// Start -
func (t *AutoTLS) Start(ctx context.Context) error {
	if t.Domain == "" {
		return errors.New("missing domain")
	}
	if t.Email == "" {
		return errors.New("missing email")
	}
	if !t.Accept {
		return errors.New("TLS ToS not accepted (--accept)")
	}
	ep := getHost(t.CAEndpoint)
	base := filepath.Join(t.getStoragePath(), ep)
	certPath := filepath.Join(base, "sites", t.Domain)
	userPath := filepath.Join(base, "users", t.Email)

	myUser := &acmeUser{}
	ok, err := myUser.load(userPath, t.Email)
	if err != nil {
		return errors.Wrapf(err, "failed to load user %s from %s", t.Email, userPath)
	}
	if !ok {
		err = myUser.create(userPath, t.Email)
		if err != nil {
			return errors.Wrapf(err, "failed to create user %s from %s", t.Email, userPath)
		}
	}

	client, err := acme.NewClient(t.CAEndpoint, myUser, acme.EC256)
	if err != nil {
		return errors.Wrapf(err, "failed to create acme client for %s", t.CAEndpoint)
	}
	client.SetHTTPAddress(t.HTTPAddress)
	client.SetTLSAddress(t.TLSAddress)

	if myUser.Registration == nil {
		reg, err := client.Register(t.Accept)
		if err != nil {
			return errors.Wrapf(err, "failed to register user %s", t.Email)
		}
		myUser.Registration = reg

		// Save the user now, since we have a registration!
		err = myUser.save(userPath)
		if err != nil {
			return errors.Wrapf(err, "failed to save user at %s", userPath)
		}
	}

	c := &cert{}
	ok, err = c.load(t.Domain, certPath)
	if err != nil {
		return errors.Wrapf(err, "failed to load cert from %s", certPath)
	}
	if !ok {
		err = c.create(t.Domain, certPath, client)
		if err != nil {
			return errors.Wrapf(err, "failed to create cert for %s", t.Domain)
		}
	}
	log.Printf("loaded cert for %s - expires %v", t.Domain, c.expiration())

	// check right away so we start with a fresh cert
	if c.needsRenewal() {
		err = c.renew(client)
		if err != nil {
			return errors.Wrap(err, "failed to renew cert")
		}
	}

	err = c.save(certPath)
	if err != nil {
		return errors.Wrap(err, "failed to save cert")
	}

	// now start the renewal cycle
	go c.renewLoop(ctx, certPath, client)

	return nil
}

func getHost(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	return u
}
