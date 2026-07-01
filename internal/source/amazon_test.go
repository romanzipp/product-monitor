package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAmazonAvailability(t *testing.T) {
	const asin = "B0TEST1234"

	inStock := `<div id="add-to-cart-button"></div>` +
		`<div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">699,00 €</span></span></div> ` + asin
	// Out of stock: the real button is gone (only a stray script reference to the
	// class remains) and id="outOfStock" is present — the bug we are fixing.
	outOfStock := `<div id="outOfStock">Derzeit nicht verfügbar.</div><script>addToCartButton</script> ` + asin
	botPage := `<html><body>Robot Check — enter the characters</body></html>`

	cases := []struct {
		name      string
		html      string
		available bool
		price     float64
	}{
		{"in stock", inStock, true, 699},
		{"out of stock", outOfStock, false, 0},
		{"bot page without asin", botPage, false, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.html))
			}))
			defer srv.Close()

			src := NewAmazon(srv.Client(), nil, []string{srv.URL + "/dp/" + asin})
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
