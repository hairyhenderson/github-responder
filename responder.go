package responder

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog"

	"net/http"

	"github.com/google/go-github/github"
	"github.com/hairyhenderson/github-responder/autotls"
	"github.com/justinas/alice"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

// Start -
func Start(ctx context.Context, opts Config, action HookHandler) (func(), error) {
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
	id, err := registerHook(ctx, client, opts)
	if err != nil {
		return nil, err
	}

	cleanup := func() {
		log := log.With().Int64("hookID", id).Logger()
		log.Info().Msg("Cleaning up webhook")
		_, err := client.Repositories.DeleteHook(ctx, opts.Owner, opts.Repo, id)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete webhook")
		}
	}

	// now listen for events
	go func() {
		c := alice.New(hlog.NewHandler(log.Logger))
		c = c.Append(
			hlog.UserAgentHandler("user_agent"),
			hlog.RefererHandler("referer"),
			hlog.MethodHandler("method"),
			hlog.URLHandler("url"),
		)
		c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			eventType := github.WebHookType(r)
			deliveryID := github.DeliveryID(r)
			l := zerolog.DebugLevel
			if status > 399 {
				l = zerolog.WarnLevel
			}
			hlog.FromRequest(r).WithLevel(l).
				Int("status", status).
				Int("size", size).
				Dur("duration", duration).
				Str("eventType", eventType).
				Str("deliveryID", deliveryID).
				Msg("-")

		}))
		http.Handle(getPath(opts.CallbackURL), c.Then(&callbackHandler{[]byte(opts.HookSecret), action}))
		http.Handle("/", c.ThenFunc(denyHandler))

		if opts.EnableTLS {
			certFile, keyFile := at.CertPaths()
			log.Info().Str("addr", opts.TLSAddress).Msg("Listening for webhook callbacks")
			err := http.ListenAndServeTLS(opts.TLSAddress, certFile, keyFile, nil)
			if err != nil {
				log.Error().Err(err).Msg("")
			}
		} else {
			log.Info().Str("addr", opts.HTTPAddress).Msg("Listening for webhook callbacks")
			err := http.ListenAndServe(opts.HTTPAddress, nil)
			if err != nil {
				log.Error().Err(err).Msg("")
			}
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
	log.Info().
		Str("hook_url", hook.GetURL()).
		Int64("hook_id", hook.GetID()).
		Str("hook_name", hook.GetName()).
		Str("callback", opts.CallbackURL).
		Msg("Registered WebHook")

	return id, nil
}

type callbackHandler struct {
	secretKey []byte
	action    HookHandler
}

func (h *callbackHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log := *hlog.FromRequest(req)
	payload, err := github.ValidatePayload(req, h.secretKey)
	if err != nil {
		log.Error().Err(err).
			Msg("invalid payload")
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	eventType := github.WebHookType(req)
	deliveryID := github.DeliveryID(req)
	log = log.With().
		Str("eventType", eventType).
		Str("deliveryID", deliveryID).Logger()
	log.Info().Msg("Incoming request")
	if eventType == "ping" {
		event, err := github.ParseWebHook(eventType, payload)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse payload")
			http.Error(resp, err.Error(), http.StatusBadRequest)
			return
		}
		ping, ok := event.(*github.PingEvent)
		if !ok {
			http.Error(resp, fmt.Sprintf("wrong event type %T", event), http.StatusBadRequest)
			return
		}
		_, err = resp.Write([]byte(*ping.Zen))
		if err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
		return
	}

	ctx := log.WithContext(req.Context())
	go h.action(ctx, eventType, deliveryID, payload)

	resp.WriteHeader(http.StatusNoContent)
}

func denyHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusNotFound)
}

// HookHandler - A function that will be executed by the callback.
//
// Payload is provided as []byte, and can be parsed with github.ParseWebHook if desired
type HookHandler func(ctx context.Context, eventType, deliveryID string, payload []byte)
