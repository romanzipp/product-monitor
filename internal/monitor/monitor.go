// Package monitor ties sources, the dedup store and the notifier together.
package monitor

import (
	"context"
	"log/slog"
	"time"

	"portasplit-monitor/internal/model"
	"portasplit-monitor/internal/notify"
	"portasplit-monitor/internal/store"
)

// Monitor periodically polls all sources and notifies on new availability.
type Monitor struct {
	sources  []model.Source
	store    *store.Store
	notifier notify.Notifier
	log      *slog.Logger
	priceMax int // 0 = unlimited
}

// New constructs a Monitor. priceMax caps accepted offer prices in whole euros
// (0 disables the limit).
func New(sources []model.Source, st *store.Store, n notify.Notifier, log *slog.Logger, priceMax int) *Monitor {
	return &Monitor{sources: sources, store: st, notifier: n, log: log, priceMax: priceMax}
}

// Run polls every interval until ctx is cancelled. The first poll runs
// immediately on start.
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

// tick performs one full poll across all sources, reconciles the dedup store
// and fires notifications for newly available items.
func (m *Monitor) tick(ctx context.Context) {
	current := make([]model.Availability, 0, 16)
	for _, src := range m.sources {
		avail, err := src.Check(ctx)
		if err != nil {
			m.log.Error("source check failed", "source", src.Name(), "err", err)
			continue
		}
		m.log.Debug("source checked", "source", src.Name(), "available", len(avail))
		for _, a := range avail {
			if !m.withinBudget(a) {
				continue
			}
			current = append(current, a)
		}
	}

	currentKeys := make(map[string]struct{}, len(current))
	for _, a := range current {
		currentKeys[a.Key] = struct{}{}
	}

	// Drop records for items that are no longer available so a future restock
	// produces a fresh notification.
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
		known, err := m.store.Exists(ctx, a.Key)
		if err != nil {
			m.log.Error("exists check failed", "key", a.Key, "err", err)
			continue
		}
		if known {
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
		m.log.Info("notified", "source", a.Source, "store", a.StoreName,
			"product", a.ProductName, "stock", a.Stock)
	}
}

// withinBudget reports whether an offer should be considered. Offers with an
// unknown price are always kept; a priceMax <= 0 disables the filter.
func (m *Monitor) withinBudget(a model.Availability) bool {
	if m.priceMax <= 0 || a.Price == nil {
		return true
	}
	if *a.Price > float64(m.priceMax) {
		if m.log.Enabled(nil, slog.LevelDebug) {
			m.log.Debug("skipping over budget",
				"key", a.Key, "price", *a.Price, "max", m.priceMax)
		}
		return false
	}
	return true
}

func sourceNames(sources []model.Source) []string {
	names := make([]string, 0, len(sources))
	for _, s := range sources {
		names = append(names, s.Name())
	}
	return names
}
