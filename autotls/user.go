package autotls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/xenolf/lego/acme"
)

// You'll need a user or account type that implements acme.User
type acmeUser struct {
	Email        string
	Registration *acme.RegistrationResource
	key          crypto.PrivateKey
	pubKey       crypto.PublicKey
}

func (u acmeUser) GetEmail() string {
	return u.Email
}
func (u acmeUser) GetRegistration() *acme.RegistrationResource {
	return u.Registration
}
func (u acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func (u acmeUser) save(dir string) error {
	err := fs.MkdirAll(dir, 0700)
	if err != nil {
		return errors.Wrapf(err, "mkdir %s", dir)
	}
	fileBase := filepath.Join(dir, u.Email)

	f, err := fs.OpenFile(fileBase+".json", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	encoder.Encode(u)

	privBytes, err := savePrivateKey(u.key)
	if err != nil {
		return err
	}
	privFile, err := fs.OpenFile(fileBase+".pem", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	_, err = privFile.Write(privBytes)
	if err != nil {
		return err
	}

	return nil
}

func (u *acmeUser) load(dir, email string) (bool, error) {
	if email == "" {
		return false, errors.New("No email specified, cannot load unnamed user")
	}
	fileBase := filepath.Join(dir, email)

	f, err := fs.OpenFile(fileBase+".json", os.O_RDONLY, 0600)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	} else if os.IsNotExist(err) {
		return false, nil
	}
	decoder := json.NewDecoder(f)
	user := &acmeUser{}
	err = decoder.Decode(user)
	if err != nil {
		return false, err
	}
	*u = *user

	privFile, err := fs.OpenFile(fileBase+".pem", os.O_RDONLY, 0600)
	if err != nil {
		return false, err
	}
	b, err := ioutil.ReadAll(privFile)
	if err != nil {
		return false, err
	}
	privKey, err := loadPrivateKey(b)
	if err != nil {
		return false, err
	}
	u.key = privKey

	log.Printf("successfully loaded user from %s", dir)
	return true, nil
}

func (u *acmeUser) create(dir, email string) error {
	if email == "" {
		return errors.New("No email specified, cannot create unnamed user")
	}
	privKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return errors.Wrapf(err, "error generating private key")
	}
	u.key = privKey
	u.pubKey = &privKey.PublicKey
	u.Email = email
	return nil
}

// loadPrivateKey loads a PEM-encoded ECC/RSA private key from an array of bytes.
func loadPrivateKey(keyBytes []byte) (crypto.PrivateKey, error) {
	keyBlock, _ := pem.Decode(keyBytes)

	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(keyBlock.Bytes)
	}

	return nil, errors.New("unknown private key type")
}

// savePrivateKey saves a PEM-encoded ECC/RSA private key to an array of bytes.
func savePrivateKey(key crypto.PrivateKey) ([]byte, error) {
	var pemType string
	var keyBytes []byte
	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		var err error
		pemType = "EC"
		keyBytes, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
	case *rsa.PrivateKey:
		pemType = "RSA"
		keyBytes = x509.MarshalPKCS1PrivateKey(key)
	}

	pemKey := pem.Block{Type: pemType + " PRIVATE KEY", Bytes: keyBytes}
	return pem.EncodeToMemory(&pemKey), nil
}
