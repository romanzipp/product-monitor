package monitor

import (
	"io"
	"log/slog"
	"testing"

	"product-monitor/internal/model"
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
