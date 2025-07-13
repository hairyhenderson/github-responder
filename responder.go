package responder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"filippo.io/mostly-harmless/cryptosource"
	"github.com/caddyserver/certmagic"
	"github.com/google/go-github/v35/github"
	"github.com/google/uuid"
	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
)

const (
	ghtokName = "GITHUB_TOKEN"
)

type repository struct {
	owner string
	name  string
}

// Responder -
type Responder struct {
	secret      string
	callbackURL string
	domain      string
	ghclient    *github.Client
	repos       []repository
	actions     []HookHandler
}

// New -
func New(repos []string, domain string, actions ...HookHandler) (*Responder, error) {
	if len(repos) == 0 {
		return nil, errors.New("must provide repo")
	}

	repositories := make([]repository, 0, len(repos))

	for _, r := range repos {
		repoParts := strings.SplitN(r, "/", 2)
		if len(repoParts) != 2 {
			return nil, fmt.Errorf("invalid repo %s - need 'owner/repo' form", r)
		}

		repositories = append(repositories, repository{repoParts[0], repoParts[1]})
	}

	// init callback URL
	callbackURL := buildCallbackURL(domain)

	// choose random secret
	//nolint:gosec // cryptosource backs with crypto/rand
	r := rand.New(cryptosource.New())
	secret := fmt.Sprintf("%x", r.Int63())

	token := os.Getenv(ghtokName)
	if token == "" {
		return nil, fmt.Errorf("GitHub API token missing - must set %s", ghtokName)
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	hc := &http.Client{Transport: &oauth2.Transport{Source: ts}}
	client := github.NewClient(hc)

	return &Responder{
		ghclient:    client,
		secret:      secret,
		repos:       repositories,
		domain:      domain,
		callbackURL: callbackURL,
		actions:     actions,
	}, nil
}

func buildCallbackURL(domain string) string {
	u := uuid.NewString()
	scheme := "https://"

	if tlsDisabled() {
		scheme = "http://"
	}

	return scheme + domain + "/gh-callback/" + u
}

// Register a new webhook with the watched repositories for the listed events. A
// cleanup function is returned when the hook is successfully registered - this
// function must be called (usually deferred), otherwise invalid webhooks will be
// left behind.
func (r *Responder) Register(ctx context.Context, events []string) (func(), error) {
	inHook := &github.Hook{
		Events: events,
		Config: map[string]interface{}{
			"url":          r.callbackURL,
			"content_type": "json",
			"secret":       r.secret,
		},
	}

	unregFuncs := make([]func(), 0, len(r.repos))

	for _, repo := range r.repos {
		owner := repo.owner
		repoName := repo.name

		hook, resp, err := r.ghclient.Repositories.CreateHook(ctx, owner, repoName, inHook)
		if err != nil {
			return nil, fmt.Errorf("failed to create hook: %w", err)
		}

		if resp.StatusCode > 299 {
			return nil, fmt.Errorf("request failed with %s", resp.Status)
		}

		id := hook.GetID()
		slog.Info("Registered WebHook",
			"hook_url", hook.GetURL(),
			"hook_id", id,
			"callback", r.callbackURL)

		unregFuncs = append(unregFuncs, func() {
			slog.Info("Cleaning up webhook", "hook_id", id)

			_, err := r.ghclient.Repositories.DeleteHook(ctx, owner, repoName, id)
			if err != nil {
				err = fmt.Errorf("failed to delete webhook: %w", err)
				slog.Error("", "error", err)
			}
		})
	}

	unregister := func() {
		for _, f := range unregFuncs {
			f()
		}
	}

	return unregister, nil
}

// slogHandler creates an HTTP middleware that adds structured logging to requests
func slogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer that captures status code and size
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: 200}

		// Call the next handler
		next.ServeHTTP(lrw, r)

		// Log the request
		duration := time.Since(start)
		eventType := github.WebHookType(r)
		deliveryID := github.DeliveryID(r)

		level := slog.LevelInfo
		if lrw.statusCode > 399 {
			level = slog.LevelWarn
		}

		slog.Log(r.Context(), level, fmt.Sprintf("%s %s - %d", r.Method, r.URL, lrw.statusCode),
			"method", r.Method,
			"url", r.URL.String(),
			"remoteAddr", r.RemoteAddr,
			"user_agent", r.Header.Get("User-Agent"),
			"referer", r.Header.Get("Referer"),
			"status", lrw.statusCode,
			"size", lrw.size,
			"duration", duration,
			"eventType", eventType,
			"deliveryID", deliveryID)
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code and response size
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.size += size

	return size, err
}

// Listen for webhooks
func (r *Responder) Listen(_ context.Context) {
	initMetrics()

	// now listen for events
	c := alice.New(slogHandler)

	http.Handle("/metrics", c.Append(filterByIP).
		Then(
			promhttp.InstrumentMetricHandler(
				MetricsRegisterer,
				promhttp.HandlerFor(MetricsGatherer,
					promhttp.HandlerOpts{},
				),
			),
		))
	http.Handle(getPath(r.callbackURL), c.Extend(instrumentHTTP("callback")).Then(r))
	http.Handle("/", c.Extend(instrumentHTTP("default")).ThenFunc(http.NotFound))

	if tlsDisabled() {
		go func() {
			slog.Info("Listening for webhook callbacks", "port", certmagic.HTTPPort)
			port := strconv.Itoa(certmagic.HTTPPort)
			srv := &http.Server{
				Addr:              ":" + port,
				ReadHeaderTimeout: 5 * time.Second,
			}

			err := srv.ListenAndServe()
			slog.Error("", "error", err)
		}()
	}

	go func() {
		slog.Info("Listening for webhook callbacks", "port", certmagic.HTTPSPort)
		err := certmagic.HTTPS([]string{r.domain}, nil)
		slog.Error("listening with certmagic", "error", err)
	}()
}

// RegisterAndListen - unlike calling `Register` and `Listen` separately, this
// will block while waiting for the context to be cancelled.
func (r *Responder) RegisterAndListen(ctx context.Context, events []string) error {
	cleanup, err := r.Register(ctx, events)
	if err != nil {
		return err
	}
	defer cleanup()

	r.Listen(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	select {
	case s := <-c:
		slog.Debug("shutting down gracefully...", "signal", s.String())
	case <-ctx.Done():
		err = ctx.Err()
		slog.Error("context cancelled", "error", err)
	}

	return err
}

func tlsDisabled() bool {
	disableTLS, err := strconv.ParseBool(os.Getenv("TLS_DISABLE"))
	if err != nil {
		return false
	}

	return disableTLS
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

func (r *Responder) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	payload, err := github.ValidatePayload(req, []byte(r.secret))
	if err != nil {
		slog.ErrorContext(ctx, "invalid payload", "error", err)
		http.Error(resp, err.Error(), http.StatusBadRequest)

		return
	}

	eventType := github.WebHookType(req)
	deliveryID := github.DeliveryID(req)
	slog.InfoContext(ctx, "Incoming request", "eventType", eventType, "deliveryID", deliveryID)

	if eventType == "ping" {
		event, err := github.ParseWebHook(eventType, payload)
		if err != nil {
			slog.ErrorContext(ctx, "failed to parse payload", "error", err)
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
			slog.ErrorContext(ctx, "failed to write response", "error", err)
		}

		return
	}

	for _, a := range r.actions {
		go a(ctx, eventType, deliveryID, payload)
	}

	resp.WriteHeader(http.StatusNoContent)
}

// HookHandler - A function that will be executed by the callback.
//
// Payload is provided as []byte, and can be parsed with github.ParseWebHook if desired
type HookHandler func(ctx context.Context, eventType, deliveryID string, payload []byte)
