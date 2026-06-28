// Package monitor ties sources, the dedup store and the notifier together.
package monitor

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"portasplit-monitor/internal/model"
	"portasplit-monitor/internal/notify"
	"portasplit-monitor/internal/store"
)

// Monitor periodically polls all sources and notifies on new availability.
type Monitor struct {
	sources       []model.Source
	store         *store.Store
	notifier      notify.Notifier
	log           *slog.Logger
	priceMax      int      // 0 = unlimited
	localPrefixes []string // in-store PLZ prefixes to keep; empty = keep all
}

func New(sources []model.Source, st *store.Store, n notify.Notifier, log *slog.Logger, priceMax int, localPrefixes []string) *Monitor {
	return &Monitor{
		sources:       sources,
		store:         st,
		notifier:      n,
		log:           log,
		priceMax:      priceMax,
		localPrefixes: localPrefixes,
	}
}

// Run polls every interval until ctx is cancelled, starting immediately.
func (m *Monitor) Run(ctx context.Context, interval time.Duration) {
	m.log.Info("monitor started", "interval", interval, "sources", sourceNames(m.sources))

	m.tick(ctx)

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			m.log.Info("monitor stopping")
			return
		case <-t.C:
			m.tick(ctx)
		}
	}
}

// tick polls all sources, reconciles the store and notifies on new items.
func (m *Monitor) tick(ctx context.Context) {
	current := make([]model.Availability, 0, 16)
	for _, src := range m.sources {
		avail, err := src.Check(ctx)
		if err != nil {
			m.log.Error("source check failed", "source", src.Name(), "err", err)
			continue
		}
		stock, minPrice := availStats(avail)
		attrs := []any{"source", src.Name(), "available", len(avail), "stock", stock}
		if minPrice != nil {
			attrs = append(attrs, "min_price", *minPrice)
		}
		m.log.Debug("source checked", attrs...)
		for _, a := range avail {
			if !m.withinBudget(a) {
				continue
			}
			if !m.isLocal(a) {
				continue
			}
			current = append(current, a)
		}
	}

	currentKeys := make(map[string]struct{}, len(current))
	for _, a := range current {
		currentKeys[a.Key] = struct{}{}
	}

	// Drop records no longer available so a future restock re-notifies.
	if existing, err := m.store.AllKeys(ctx); err != nil {
		m.log.Error("loading tracked keys failed", "err", err)
	} else {
		for _, k := range existing {
			if _, ok := currentKeys[k]; !ok {
				if err := m.store.Delete(ctx, k); err != nil {
					m.log.Error("delete stale failed", "key", k, "err", err)
				} else {
					m.log.Info("availability gone, will re-notify on restock", "key", k)
				}
			}
		}
	}

	if len(current) == 0 {
		m.log.Info("no availability found this cycle")
		return
	}

	for _, a := range current {
		prevPrice, known, err := m.store.Lookup(ctx, a.Key)
		if err != nil {
			m.log.Error("lookup failed", "key", a.Key, "err", err)
			continue
		}

		if known {
			// Re-notify only when the price dropped below the last seen one.
			if a.Price != nil && prevPrice != nil && *a.Price < *prevPrice {
				if err := m.notifier.Notify(ctx, a); err != nil {
					// Don't update the price: retry the drop on the next cycle.
					m.log.Error("notify failed", "key", a.Key, "err", err)
					continue
				}
				m.log.Info("price drop notified", "key", a.Key, "old", *prevPrice, "new", *a.Price)
			}
			if err := m.store.Touch(ctx, a); err != nil {
				m.log.Error("touch failed", "key", a.Key, "err", err)
			}
			continue
		}

		if err := m.notifier.Notify(ctx, a); err != nil {
			// Don't record: retry on the next cycle.
			m.log.Error("notify failed", "key", a.Key, "err", err)
			continue
		}
		if err := m.store.Record(ctx, a); err != nil {
			m.log.Error("record failed", "key", a.Key, "err", err)
		}
		m.log.Info("notified", "source", a.Source, "store", a.StoreName, "product", a.ProductName, "stock", a.Stock)
	}
}

// isLocal reports whether an availability passes the local-store filter. Online
// always passes; in-store passes only when its PLZ matches a configured prefix.
func (m *Monitor) isLocal(a model.Availability) bool {
	if a.Channel != model.ChannelInStore || len(m.localPrefixes) == 0 {
		return true
	}
	for _, p := range m.localPrefixes {
		if strings.HasPrefix(a.PLZ, p) {
			return true
		}
	}
	m.log.Debug("skipping non-local in-store result", "store", a.StoreName, "plz", a.PLZ, "prefixes", m.localPrefixes)
	return false
}

// withinBudget keeps offers with unknown price; priceMax <= 0 disables the filter.
func (m *Monitor) withinBudget(a model.Availability) bool {
	if m.priceMax <= 0 || a.Price == nil {
		return true
	}
	if *a.Price > float64(m.priceMax) {
		m.log.Debug("skipping over budget", "key", a.Key, "price", *a.Price, "max", m.priceMax)
		return false
	}
	return true
}

// availStats sums the stock and finds the lowest known price across results.
func availStats(avail []model.Availability) (stock int, minPrice *float64) {
	for _, a := range avail {
		stock += a.Stock
		if a.Price != nil && (minPrice == nil || *a.Price < *minPrice) {
			p := *a.Price
			minPrice = &p
		}
	}
	return stock, minPrice
}

func sourceNames(sources []model.Source) []string {
	names := make([]string, 0, len(sources))
	for _, s := range sources {
		names = append(names, s.Name())
	}
	return names
}
