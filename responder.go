package responder

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mholt/certmagic"

	"github.com/rs/zerolog"

	"net/http"

	"github.com/google/go-github/v20/github"
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

	certmagic.CA = opts.CAEndpoint
	certmagic.HTTPPort = opts.HTTPPort
	certmagic.HTTPSPort = opts.HTTPSPort

	initMetrics()

	// now listen for events
	go func() {
		c := alice.New(hlog.NewHandler(log.Logger))
		c = c.Append(
			hlog.UserAgentHandler("user_agent"),
			hlog.RefererHandler("referer"),
			hlog.MethodHandler("method"),
			hlog.URLHandler("url"),
			hlog.RemoteAddrHandler("remoteAddr"),
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
				Msgf("%s %s - %d", r.Method, r.URL, status)
		}))
		http.Handle("/metrics", c.Append(filterByIP).Extend(instrumentHTTP("metrics")).Then(promhttp.Handler()))
		http.Handle(getPath(opts.CallbackURL), c.Extend(instrumentHTTP("callback")).Then(&callbackHandler{[]byte(opts.HookSecret), action}))
		http.Handle("/", c.Extend(instrumentHTTP("default")).ThenFunc(denyHandler))

		if opts.EnableTLS {
			log.Info().Int("port", opts.HTTPSPort).Msg("Listening for webhook callbacks")
			err := certmagic.HTTPS([]string{opts.Domain}, nil)
			if err != nil {
				log.Error().Err(err).Msg("")
			}
		} else {
			log.Info().Int("port", opts.HTTPPort).Msg("Listening for webhook callbacks")
			port := strconv.Itoa(opts.HTTPPort)
			err := http.ListenAndServe(":"+port, nil)
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
