package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dmesha3/elgon"
)

type metricKey struct {
	method string
	route  string
	status int
}

// Metrics holds HTTP request counters and latency totals.
type Metrics struct {
	mu                sync.Mutex
	requestsTotal     map[metricKey]uint64
	requestDurationNs map[metricKey]uint64
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestsTotal:     make(map[metricKey]uint64),
		requestDurationNs: make(map[metricKey]uint64),
	}
}

// Middleware records request count and latency by method/route/status.
func (m *Metrics) Middleware() elgon.Middleware {
	return func(next elgon.HandlerFunc) elgon.HandlerFunc {
		return func(c *elgon.Ctx) error {
			start := time.Now()
			rw := &statusWriter{ResponseWriter: c.Writer, status: http.StatusOK}
			c.Writer = rw

			err := next(c)
			if err != nil {
				if he, ok := err.(*elgon.HTTPError); ok {
					rw.status = he.Status
				} else {
					rw.status = http.StatusInternalServerError
				}
			}

			route := c.RoutePattern()
			if route == "" {
				route = c.Request.URL.Path
			}
			m.record(c.Request.Method, route, rw.status, time.Since(start))
			return err
		}
	}
}

func (m *Metrics) record(method, route string, status int, d time.Duration) {
	k := metricKey{method: method, route: route, status: status}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestsTotal[k]++
	m.requestDurationNs[k] += uint64(d)
}

// Handler serves metrics in Prometheus text format.
func (m *Metrics) Handler() elgon.HandlerFunc {
	return func(c *elgon.Ctx) error {
		c.Writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, err := c.Writer.Write([]byte(m.Export()))
		return err
	}
}

// RegisterRoute registers a GET /metrics endpoint on an app.
func (m *Metrics) RegisterRoute(app *elgon.App, path string) {
	if path == "" {
		path = "/metrics"
	}
	app.GET(path, m.Handler())
}

func (m *Metrics) Export() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := make([]metricKey, 0, len(m.requestsTotal))
	for k := range m.requestsTotal {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		return keys[i].status < keys[j].status
	})

	var b strings.Builder
	b.WriteString("# HELP elgon_http_requests_total Total HTTP requests.\n")
	b.WriteString("# TYPE elgon_http_requests_total counter\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "elgon_http_requests_total{method=\"%s\",route=\"%s\",status=\"%d\"} %d\n",
			escape(k.method), escape(k.route), k.status, m.requestsTotal[k])
	}
	b.WriteString("# HELP elgon_http_request_duration_seconds_total Total request duration seconds.\n")
	b.WriteString("# TYPE elgon_http_request_duration_seconds_total counter\n")
	for _, k := range keys {
		seconds := float64(m.requestDurationNs[k]) / float64(time.Second)
		fmt.Fprintf(&b, "elgon_http_request_duration_seconds_total{method=\"%s\",route=\"%s\",status=\"%d\"} %.6f\n",
			escape(k.method), escape(k.route), k.status, seconds)
	}
	return b.String()
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
