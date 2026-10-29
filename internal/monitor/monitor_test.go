package monitor

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"product-monitor/internal/model"
	"product-monitor/internal/store"
)

func TestIsLocal(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name     string
		prefixes []string
		a        model.Availability
		want     bool
	}{
		{"online always passes", []string{"36"}, model.Availability{Channel: model.ChannelOnline}, true},
		{"no prefixes keeps everything", nil, model.Availability{Channel: model.ChannelInStore, PLZ: "13127"}, true},
		{"local in-store passes", []string{"36"}, model.Availability{Channel: model.ChannelInStore, PLZ: "36100"}, true},
		{"far in-store filtered", []string{"36"}, model.Availability{Channel: model.ChannelInStore, PLZ: "13127"}, false},
		{"multiple prefixes", []string{"36", "97"}, model.Availability{Channel: model.ChannelInStore, PLZ: "97070"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := New(nil, nil, nil, log, tc.prefixes, nil)
			if got := m.isLocal(tc.a); got != tc.want {
				t.Errorf("isLocal=%v, want %v", got, tc.want)
			}
		})
	}
}

type fakeSource struct{ avail []model.Availability }

func (f fakeSource) Name() string { return "fake" }

func (f fakeSource) Check(context.Context) ([]model.Availability, error) { return f.avail, nil }

type fakeNotifier struct{}

func (fakeNotifier) Notify(context.Context, model.Availability) error { return nil }

func TestTickKeepsOtherProductsRows(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	a := Product{Name: "A", Sources: []model.Source{fakeSource{[]model.Availability{{Key: "a1", ProductName: "A"}}}}}
	b := Product{Name: "B", Sources: []model.Source{fakeSource{[]model.Availability{{Key: "b1", ProductName: "B"}}}}}
	m := New([]Product{a, b}, st, fakeNotifier{}, log, nil, nil)

	m.tick(ctx, a)
	m.tick(ctx, b)

	a.Sources = []model.Source{fakeSource{}}
	m.tick(ctx, a)

	if _, known, _ := st.Lookup(ctx, "a1"); known {
		t.Error("a1 should be dropped once A no longer reports it")
	}
	if _, known, _ := st.Lookup(ctx, "b1"); !known {
		t.Error("b1 must survive a tick of product A")
	}
}
