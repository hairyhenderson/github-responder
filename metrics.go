package responder

import (
	"net"
	"net/http"
	"strings"

	"github.com/justinas/alice"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var (
	ns            = "http"
	httpLabels    = []string{"handler", "code", "method"}
	durBuckets    = []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10}
	sumObjectives = map[float64]float64{0.1: 0.5, 0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 0.999: 0.0001}
	observers     = map[string]prometheus.ObserverVec{
		"durationHistogram": prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "request_duration_seconds",
			Help:      "A histogram of latencies for requests.",
			Buckets:   durBuckets,
		}, httpLabels),
		"durationSummary": prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  ns,
			Name:       "request_duration_quantile_seconds",
			Help:       "A summary of latencies for requests.",
			Objectives: sumObjectives,
		}, httpLabels),
		"responseSizeHistogram": prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "response_size_bytes",
			Help:      "A histogram of response sizes for requests.",
			Buckets:   []float64{200, 500, 900, 1500},
		}, httpLabels),
		"responseSizeSummary": prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  ns,
			Name:       "response_size_quantile_bytes",
			Help:       "A summary of response sizes for requests.",
			Objectives: sumObjectives,
		}, httpLabels),
		"requestSizeHistogram": prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "request_size_bytes",
			Help:      "A histogram of request sizes for requests.",
			Buckets:   []float64{200, 500, 900, 1500},
		}, httpLabels),
		"requestSizeSummary": prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  ns,
			Name:       "request_size_quantile_bytes",
			Help:       "A summary of request sizes for requests.",
			Objectives: sumObjectives,
		}, httpLabels)}
)

func initMetrics() {
	o := []prometheus.Collector{}
	for _, m := range observers {
		o = append(o, m)
	}
	prometheus.MustRegister(o...)
}

func instrumentHTTP(handler string) alice.Chain {
	l := prometheus.Labels{"handler": handler}
	chain := alice.New()
	for k, v := range observers {
		if strings.HasPrefix(k, "duration") {
			chain = chain.Append(func(next http.Handler) http.Handler {
				return promhttp.InstrumentHandlerDuration(v.MustCurryWith(l), next)
			})
		} else if strings.HasPrefix(k, "request") {
			chain = chain.Append(func(next http.Handler) http.Handler {
				return promhttp.InstrumentHandlerRequestSize(v.MustCurryWith(l), next)
			})
		} else if strings.HasPrefix(k, "response") {
			chain = chain.Append(func(next http.Handler) http.Handler {
				return promhttp.InstrumentHandlerResponseSize(v.MustCurryWith(l), next)
			})
		} else {
			panic(errors.Errorf("bad metric name %s", k))
		}
	}
	return chain
}

func filterByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			ip := net.ParseIP(host)
			if isAllowed(ip) {
				next.ServeHTTP(resp, req)
				return
			}
		}

		log.Warn().Str("remoteAddr", req.RemoteAddr).Msg("bad remoteAddr - rejecting")
		resp.WriteHeader(http.StatusNotFound)
	})
}

func isAllowed(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLinkLocalUnicast() || ip.IsLoopback() {
		return true
	}

	_, cidr, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't parse the CIDR")
	}
	return cidr.Contains(ip)
}
