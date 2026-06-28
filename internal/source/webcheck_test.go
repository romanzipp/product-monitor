package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"portasplit-monitor/internal/model"
)

func newWebCheckTestServer(t *testing.T, html string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
}

func TestWebCheckAvailability(t *testing.T) {
	in := []string{"in den warenkorb"}
	out := []string{"nicht verfügbar", "ausverkauft"}

	cases := []struct {
		name      string
		html      string
		available bool
	}{
		{"in stock", `<button>In den Warenkorb</button> Lieferung ab 09.07.2026`, true},
		{"out of stock wins over add-to-cart", `<button>In den Warenkorb</button> Artikel nicht verfügbar`, false},
		{"sold out", `<div>Ausverkauft</div>`, false},
		{"no markers at all", `<div>irgendwas</div>`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newWebCheckTestServer(t, tc.html)
			defer srv.Close()

			src := NewWebCheck("mediamarkt", srv.Client(), nil, srv.URL,
				"MediaMarkt", "Midea PortaSplit", model.ChannelOnline, in, out)
			got, err := src.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if available := len(got) > 0; available != tc.available {
				t.Fatalf("available=%v, want %v (results=%+v)", available, tc.available, got)
			}
			if tc.available && got[0].Channel != model.ChannelOnline {
				t.Errorf("channel=%s, want online", got[0].Channel)
			}
		})
	}
}
