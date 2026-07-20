package source

import (
	"context"
	"testing"
)

func TestShopifyCollection(t *testing.T) {
	// Two products: one sold out, one with an available variant.
	body := `{"products":[
		{"id":1,"title":"Bag A","handle":"bag-a","variants":[{"available":false,"price":"80.00"}]},
		{"id":2,"title":"Bag B","handle":"bag-b","variants":[
			{"available":false,"price":"90.00"},
			{"available":true,"price":"85.00"}
		]}
	]}`
	srv := newPageServer(t, body)
	defer srv.Close()

	src := NewShopifyCollection(srv.Client(), nil, []string{srv.URL + "/collections/bags"}, "Second Catch")
	got, err := src.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 available product, got %d: %+v", len(got), got)
	}
	a := got[0]
	if a.StoreName != "Bag B" {
		t.Errorf("StoreName=%q, want Bag B", a.StoreName)
	}
	if a.Price == nil || *a.Price != 85 {
		t.Errorf("price=%v, want 85 (lowest available variant)", a.Price)
	}
	if a.URL != srv.URL+"/products/bag-b" {
		t.Errorf("URL=%q, want .../products/bag-b", a.URL)
	}
	if a.Key != "shopify-collection:2" {
		t.Errorf("Key=%q, want shopify-collection:2", a.Key)
	}
}
