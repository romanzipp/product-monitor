// Package metrics exposes per-source monitor state in Prometheus text format.
// It is hand-rolled to avoid a client library dependency for a handful of gauges.
package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Metrics holds the latest observed state per source.
type Metrics struct {
	mu  sync.Mutex
	src map[string]*sourceState
}

type sourceState struct {
	lastCheck     int64
	success       int
	available     int
	stock         int
	minPrice      float64
	hasPrice      bool
	checksSuccess int64
	checksError   int64
	notifications int64
}

func New() *Metrics {
	return &Metrics{src: make(map[string]*sourceState)}
}

// get returns the state for a source, creating it on first use. Caller holds mu.
func (m *Metrics) get(source string) *sourceState {
	s := m.src[source]
	if s == nil {
		s = &sourceState{}
		m.src[source] = s
	}
	return s
}

// ObserveCheck records the outcome of one source poll.
func (m *Metrics) ObserveCheck(source string, ok bool, available, stock int, minPrice *float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.get(source)
	s.lastCheck = time.Now().Unix()
	if !ok {
		s.success = 0
		s.checksError++
		return
	}
	s.success = 1
	s.checksSuccess++
	s.available = available
	s.stock = stock
	if minPrice != nil {
		s.minPrice = *minPrice
		s.hasPrice = true
	} else {
		s.hasPrice = false
	}
}

// ObserveNotification records a notification sent for a source.
func (m *Metrics) ObserveNotification(source string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.get(source).notifications++
}

// Handler serves the metrics in Prometheus text exposition format.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		m.write(w)
	})
}

func (m *Metrics) write(w io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	names := make([]string, 0, len(m.src))
	for n := range m.src {
		names = append(names, n)
	}
	sort.Strings(names)

	gauge(w, "product_source_up", "Whether the last check for a source succeeded (1) or failed (0)")
	for _, n := range names {
		sample(w, "product_source_up", n, float64(m.src[n].success))
	}

	gauge(w, "product_source_last_check_timestamp_seconds", "Unix time of the last check per source")
	for _, n := range names {
		sample(w, "product_source_last_check_timestamp_seconds", n, float64(m.src[n].lastCheck))
	}

	gauge(w, "product_source_available", "Number of in-stock offerings from the last check")
	for _, n := range names {
		sample(w, "product_source_available", n, float64(m.src[n].available))
	}

	gauge(w, "product_source_stock", "Total units in stock from the last check")
	for _, n := range names {
		sample(w, "product_source_stock", n, float64(m.src[n].stock))
	}

	gauge(w, "product_source_min_price_euros", "Lowest known price from the last check, when available")
	for _, n := range names {
		if m.src[n].hasPrice {
			sample(w, "product_source_min_price_euros", n, m.src[n].minPrice)
		}
	}

	counter(w, "product_source_checks_total", "Total source checks by result")
	for _, n := range names {
		sampleLabels(w, "product_source_checks_total", n, "success", float64(m.src[n].checksSuccess))
		sampleLabels(w, "product_source_checks_total", n, "error", float64(m.src[n].checksError))
	}

	counter(w, "product_source_notifications_total", "Total notifications sent per source")
	for _, n := range names {
		sample(w, "product_source_notifications_total", n, float64(m.src[n].notifications))
	}
}

func gauge(w io.Writer, name, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", name, help, name)
}

func counter(w io.Writer, name, help string) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", name, help, name)
}

func sample(w io.Writer, name, source string, v float64) {
	fmt.Fprintf(w, "%s{source=%q} %g\n", name, source, v)
}

func sampleLabels(w io.Writer, name, source, result string, v float64) {
	fmt.Fprintf(w, "%s{source=%q,result=%q} %g\n", name, source, result, v)
}
