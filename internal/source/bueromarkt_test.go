package source

import (
	"context"
	"testing"
)

func TestBueromarktAvailability(t *testing.T) {
	const id = "11647"

	inStock := `<span class="product_price mainPrice" data-price-net="1.499,99 €" data-price-gross="1.784,99 €">1.499,99</span>` +
		`<div class="product-status product-delivery-status immediateDelivery">sofort lieferbar&sup1;</div> sku ` + id
	outOfStock := `<div class="product-status product-order-status">derzeit nicht verfügbar</div>` +
		`<a id="notify-product-` + id + `">benachrichtigen</a> ` + id
	// Soft-404: someone else's page without our product id.
	soft404 := `<div class="product-status product-delivery-status immediateDelivery">sofort lieferbar</div> 99999`

	cases := []struct {
		name      string
		html      string
		available bool
		price     float64
	}{
		{"in stock", inStock, true, 1784.99},
		{"out of stock", outOfStock, false, 0},
		{"soft 404 without token", soft404, false, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newPageServer(t, tc.html)
			defer srv.Close()

			src := NewBueromarkt(srv.Client(), nil, []string{srv.URL + "/klimageraet,p-" + id + ".html"})
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
