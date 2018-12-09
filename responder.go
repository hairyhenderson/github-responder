package responder

import (
	"context"
	"net/url"

	"log"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/hairyhenderson/github-responder/autotls"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// Start -
func Start(ctx context.Context, opts Config, action ActionFunc) (func(), error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: opts.GitHubToken},
	)
	hc := &http.Client{Transport: &oauth2.Transport{Source: ts}}
	client := github.NewClient(hc)

	// TLS stuff
	var at *autotls.AutoTLS
	if opts.EnableTLS {
		at = autotls.New(opts.Domain, opts.Email)
		at.HTTPAddress = opts.HTTPAddress
		at.TLSAddress = opts.TLSAddress
		at.CAEndpoint = opts.CAEndpoint
		at.Accept = opts.Accept
		at.StoragePath = opts.StoragePath

		err := at.Start(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to autotls.Start")
		}
	}

	// Register the webhook with GitHub
	log.Println("Registering WebHook...")
	id, err := registerHook(ctx, client, opts)
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		log.Printf("deleting hook %d", id)
		client.Repositories.DeleteHook(ctx, opts.Owner, opts.Repo, id)
	}

	// now listen for events
	go func() {
		http.HandleFunc(getPath(opts.CallbackURL), handleCallback(opts.HookSecret, action))
		http.HandleFunc("/", denyHandler)

		if opts.EnableTLS {
			certFile, keyFile := at.CertPaths()
			log.Printf("Listening for webhook callbacks on %s", opts.TLSAddress)
			http.ListenAndServeTLS(opts.TLSAddress, certFile, keyFile, nil)
		} else {
			log.Printf("Listening for webhook callbacks on %s", opts.HTTPAddress)
			http.ListenAndServe(opts.HTTPAddress, nil)
		}
	}()

	return cleanup, nil
}

func getPath(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	if parsed.Path != "" {
		return parsed.Path
	}
	return u
}

func registerHook(ctx context.Context, client *github.Client, opts Config) (int64, error) {
	hook := &github.Hook{
		Events: opts.Events,
		Config: map[string]interface{}{
			"url":          opts.CallbackURL,
			"content_type": "json",
			"secret":       opts.HookSecret,
		},
	}

	hook, resp, err := client.Repositories.CreateHook(ctx, opts.Owner, opts.Repo, hook)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create hook")
	}
	if resp.StatusCode > 299 {
		return 0, errors.Errorf("request failed with %s", resp.Status)
	}
	id := hook.GetID()
	log.Printf("created hook, URL is %s", hook.GetURL())
	log.Printf("callback is at %s", opts.CallbackURL)

	return id, nil
}

func handleCallback(secret string, action ActionFunc) func(resp http.ResponseWriter, req *http.Request) {
	secretKey := []byte(secret)
	return func(resp http.ResponseWriter, req *http.Request) {
		log.Printf("Incoming request at %s", req.URL)
		payload, err := github.ValidatePayload(req, secretKey)
		if err != nil {
			log.Println(err)
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}

		eventType := github.WebHookType(req)
		deliveryID := github.DeliveryID(req)
		go action(eventType, deliveryID, payload)

		resp.WriteHeader(204)
	}
}

func denyHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusNotFound)
}

// ActionFunc - A function that will be executed by the callback
//
// Payload is provided as []byte, and can be parsed with github.ParseWebHook if desired
type ActionFunc func(eventType, deliveryID string, payload []byte)
