package autotls

import (
	"fmt"
)

// AutoTLS sets up reasonable defaults
func ExampleNew() {
	autoTLS := New("example.com", "me@example.com")

	fmt.Printf("%#v", autoTLS)

	// Output:
	// &autotls.AutoTLS{Domain:"example.com", Email:"me@example.com", HTTPAddress:":80", TLSAddress:":443", StoragePath:"~/.lego", CAEndpoint:"https://acme-v02.api.letsencrypt.org/directory"}
}

// Settings can be altered to change ports
func ExampleNew_custom_ports() {
	autoTLS := New("example.com", "me@example.com")
	autoTLS.HTTPAddress = ":8080"
	autoTLS.TLSAddress = ":8443"

	fmt.Printf("%#v", autoTLS)

	// Output:
	// &autotls.AutoTLS{Domain:"example.com", Email:"me@example.com", HTTPAddress:":8080", TLSAddress:":8443", StoragePath:"~/.lego", CAEndpoint:"https://acme-v02.api.letsencrypt.org/directory"}
}

// Use the LetsEncryptStagingURL for testing so you don't hit rate limits
func ExampleNew_staging() {
	autoTLS := New("example.com", "me@example.com")
	autoTLS.CAEndpoint = LetsEncryptStagingURL

	fmt.Printf("%#v", autoTLS)

	// Output:
	// &autotls.AutoTLS{Domain:"example.com", Email:"me@example.com", HTTPAddress:":80", TLSAddress:":443", StoragePath:"~/.lego", CAEndpoint:"https://acme-staging-v02.api.letsencrypt.org/directory"}
}
