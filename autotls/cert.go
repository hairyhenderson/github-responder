package autotls

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/xenolf/lego/acme"
)

const (
	// certs are renewed if they're expiring in <30 days
	renewBefore = (24 * time.Hour) * 30

	// certs are checked for renewal every 12 hours
	renewInterval = 12 * time.Hour
)

var fs = afero.NewOsFs()

// cert convenience wrapper
type cert struct {
	certResource *acme.CertificateResource
}

func (c cert) save(dir string) error {
	domain := c.certResource.Domain
	fileBase := filepath.Join(dir, domain)

	err := fs.MkdirAll(dir, 0700)
	if err != nil {
		return errors.Wrapf(err, "mkdir %s", dir)
	}

	certFile, err := fs.OpenFile(fileBase+".crt", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	_, err = certFile.Write(c.certResource.Certificate)
	if err != nil {
		return err
	}
	privFile, err := fs.OpenFile(fileBase+".key", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	_, err = privFile.Write(c.certResource.PrivateKey)
	if err != nil {
		return err
	}
	f, err := fs.OpenFile(fileBase+".json", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(c.certResource)
	if err != nil {
		return err
	}

	return nil
}

func loadCertResource(filename string) (*acme.CertificateResource, error) {
	f, err := fs.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if os.IsNotExist(err) {
		return nil, nil
	}
	decoder := json.NewDecoder(f)
	certResource := &acme.CertificateResource{}
	err = decoder.Decode(certResource)
	if err != nil {
		return nil, err
	}
	return certResource, nil
}

func loadFile(filename string) ([]byte, error) {
	f, err := fs.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "failed to load %s", filename)
	} else if os.IsNotExist(err) {
		return nil, nil
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *cert) load(domain, dir string) (bool, error) {
	fileBase := filepath.Join(dir, domain)

	certResource, err := loadCertResource(fileBase + ".json")
	if err != nil {
		return false, err
	}
	if certResource == nil {
		return false, nil
	}

	// sanity check
	if certResource.Domain != domain {
		return false, errors.Errorf("%s contains wrong domain %s (expected %s)", dir, certResource.Domain, domain)
	}

	b, err := loadFile(fileBase + ".crt")
	if err != nil {
		return false, err
	}
	if b == nil {
		return false, nil
	}
	certResource.Certificate = b

	b, err = loadFile(fileBase + ".key")
	if err != nil {
		return false, err
	}
	if b == nil {
		return false, nil
	}
	certResource.PrivateKey = b

	c.certResource = certResource
	log.Debug().Str("path", dir).Msg("successfully loaded cert")
	return true, nil
}

func (c *cert) create(domain, dir string, client *acme.Client) error {
	certResource, err := client.ObtainCertificate([]string{domain}, true, nil, false)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain cert for %s", domain)
	}
	c.certResource = certResource

	return nil
}

func (c cert) needsRenewal() bool {
	if c.certResource != nil {
		expiration, err := acme.GetPEMCertExpiration(c.certResource.Certificate)
		if err != nil {
			log.Error().Err(err).Msg("error while checking expiration")
			return true
		}
		timeLeft := expiration.Sub(time.Now().UTC())
		if timeLeft < renewBefore {
			return true
		}
	}
	return false
}

func (c cert) expiration() time.Time {
	expiration, err := acme.GetPEMCertExpiration(c.certResource.Certificate)
	if err != nil {
		log.Error().Err(err).Msg("error while checking expiration")
	}
	return expiration
}

func (c *cert) renew(client *acme.Client) error {
	newCR, err := client.RenewCertificate(*c.certResource, true, false)
	if err != nil {
		return err
	}
	c.certResource = newCR
	return nil
}

func (c *cert) checkAndRenew(dir string, client *acme.Client) error {
	if c.needsRenewal() {
		err := c.renew(client)
		if err != nil {
			return err
		}
	}

	return c.save(dir)
}

func (c *cert) renewLoop(ctx context.Context, dir string, client *acme.Client) {
	tick := time.NewTicker(renewInterval)
	for {
		select {
		case <-tick.C:
			log.Debug().Dur("interval", renewInterval).Msg("checking if cert needs renewal")
			err := c.checkAndRenew(dir, client)
			if err != nil {
				log.Error().
					Err(err).
					Msg("error while renewing cert")
			}
		case <-ctx.Done():
			log.Error().
				Err(ctx.Err()).
				Msg("context interrupted during renew loop")
			return
		}
	}
}
