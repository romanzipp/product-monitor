package source

import (
	"context"
	"testing"
)

func TestBobsElektroAvailability(t *testing.T) {
	cases := []struct {
		name      string
		html      string
		available bool
		price     float64
	}{
		{
			"in stock green",
			`<div class="deliverytime deliverytime-green">sofort lieferbar</div><div class="price"><span>1.849,00&nbsp;€</span></div>`,
			true, 1849.00,
		},
		{
			"available yellow",
			`<div class="deliverytime deliverytime-yellow">Lieferzeit 5-10 Werktage</div><div class="price">999,00 €</div>`,
			true, 999.00,
		},
		{
			// First (main) badge is red; a later green cross-sell must not flip it.
			"out of stock red wins",
			`<div class="deliverytime deliverytime-red">nicht mehr lieferbar</div><div class="price">1.849,00 €</div>` +
				`<div class="deliverytime deliverytime-green">sofort lieferbar</div>`,
			false, 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newPageServer(t, tc.html)
			defer srv.Close()

			src := NewBobsElektro(srv.Client(), nil, []string{srv.URL + "/x/midea-portasplit.html"})
			got, err := src.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if available := len(got) > 0; available != tc.available {
				t.Fatalf("available=%v, want %v (%+v)", available, tc.available, got)
			}
			if tc.available {
				if got[0].Price == nil || *got[0].Price != tc.price {
					t.Errorf("price=%v, want %v", got[0].Price, tc.price)
				}
			}
		})
	}
}
