package source

import (
	"context"
	"testing"
)

func TestGrzAvailability(t *testing.T) {
	const id = "1768"

	base := `<meta itemprop="productID" content="` + id + `"/>` +
		`<meta itemprop="price" content="999.00"/>` +
		`<link itemprop="availability" href="https://schema.org/LimitedAvailability"/>`

	cases := []struct {
		name      string
		html      string
		available bool
		preOrder  bool
	}{
		{"in stock short delivery", base + `<span class="delivery--text">Lieferzeit 5 Werktage</span>`, true, false},
		{"pre-order long delivery", base + `<span class="delivery--text">Lieferzeit 63 Werktage</span>`, true, true},
		{"no delivery info", base, false, false},
		{"sold out", base + `Lieferzeit 5 Werktage nicht mehr verfügbar`, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newPageServer(t, tc.html)
			defer srv.Close()

			src := NewGrz(srv.Client(), nil, []string{srv.URL + "/klima/mobile/" + id + "/slug"})
			got, err := src.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if available := len(got) > 0; available != tc.available {
				t.Fatalf("available=%v, want %v (%+v)", available, tc.available, got)
			}
			if tc.available {
				if got[0].PreOrder != tc.preOrder {
					t.Errorf("preOrder=%v, want %v", got[0].PreOrder, tc.preOrder)
				}
				if got[0].Price == nil || *got[0].Price != 999 {
					t.Errorf("price=%v, want 999", got[0].Price)
				}
			}
		})
	}
}
