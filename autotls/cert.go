package autotls

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
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

func (c *cert) load(domain, dir string) (bool, error) {
	fileBase := filepath.Join(dir, domain)

	f, err := fs.OpenFile(fileBase+".json", os.O_RDONLY, 0600)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	} else if os.IsNotExist(err) {
		return false, nil
	}
	decoder := json.NewDecoder(f)
	certResource := &acme.CertificateResource{}
	err = decoder.Decode(certResource)
	if err != nil {
		return false, err
	}

	// sanity check
	if certResource.Domain != domain {
		return false, errors.Errorf("%s contains wrong domain %s (expected %s)", dir, certResource.Domain, domain)
	}

	certFile, err := fs.OpenFile(fileBase+".crt", os.O_RDONLY, 0600)
	if err != nil && !os.IsNotExist(err) {
		return false, errors.Wrapf(err, "missing .crt file")
	} else if os.IsNotExist(err) {
		return false, nil
	}
	b, err := ioutil.ReadAll(certFile)
	if err != nil {
		return false, err
	}
	certResource.Certificate = b

	privFile, err := fs.OpenFile(fileBase+".key", os.O_RDONLY, 0600)
	if err != nil && !os.IsNotExist(err) {
		return false, errors.Wrapf(err, "missing .key file")
	} else if os.IsNotExist(err) {
		return false, nil
	}
	b, err = ioutil.ReadAll(privFile)
	if err != nil {
		return false, err
	}
	certResource.PrivateKey = b
	c.certResource = certResource
	log.Printf("successfully loaded cert from %s", dir)
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
			log.Printf("error while checking expiration: %v", err)
			return true
		}
		timeLeft := expiration.Sub(time.Now().UTC())
		log.Printf("cert has %v remaining", timeLeft)
		if timeLeft < renewBefore {
			log.Println("will renew")
			return true
		}
	}
	return false
}

func (c cert) expiration() time.Time {
	expiration, err := acme.GetPEMCertExpiration(c.certResource.Certificate)
	if err != nil {
		log.Printf("error while checking expiration: %v", err)
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
	log.Println("checkAndRenew")
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
			err := c.checkAndRenew(dir, client)
			if err != nil {
				log.Printf("error while renewing cert: %v", err)
			}
		case <-ctx.Done():
			log.Println(ctx.Err())
			return
		}
	}
}
