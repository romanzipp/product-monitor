package source

import (
	"context"
	"testing"
)

func TestMediaDealAvailability(t *testing.T) {
	const id = "19421"
	base := `<meta itemprop="price" content="1999.00"/> "productid":` + id + ` `

	cases := []struct {
		name      string
		html      string
		available bool
	}{
		{"in stock", base + `<link itemprop="availability" href="http://schema.org/InStock"/>`, true},
		// LimitedAvailability is out of stock on this shop.
		{"limited availability is oos", base + `<link itemprop="availability" href="http://schema.org/LimitedAvailability"/> lieferzeit anfragen`, false},
		{"out of stock", base + `<link itemprop="availability" href="http://schema.org/OutOfStock"/>`, false},
		{"soft 404 without token", `<link itemprop="availability" href="http://schema.org/InStock"/> 99999`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newPageServer(t, tc.html)
			defer srv.Close()

			src := NewMediaDeal(srv.Client(), nil, []string{srv.URL + "/klimageraete/" + id + "/midea-porta-split"})
			got, err := src.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if available := len(got) > 0; available != tc.available {
				t.Fatalf("available=%v, want %v (%+v)", available, tc.available, got)
			}
			if tc.available && (got[0].Price == nil || *got[0].Price != 1999) {
				t.Errorf("price=%v, want 1999", got[0].Price)
			}
		})
	}
}
