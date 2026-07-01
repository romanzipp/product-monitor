package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePrice(t *testing.T) {
	cases := []struct {
		name string
		html string
		want float64 // 0 means nil expected
	}{
		{"microdata", `<meta itemprop="price" content="39.89" />`, 39.89},
		{"json number", `"price":2949.99`, 2949.99},
		{"skips label, finds number", `"price":"preis exkl. versand" "price":799.00`, 799.00},
		{"german comma", `"price":"799,00"`, 799.00},
		{"german thousands", `"price":"2.949,99"`, 2949.99},
		{"none", `<div>no price here</div>`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePrice(tc.html)
			if tc.want == 0 {
				if got != nil {
					t.Fatalf("want nil, got %v", *got)
				}
				return
			}
			if got == nil || *got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestSchemaPreOrder(t *testing.T) {
	const token = mmToken
	cases := []struct {
		name      string
		html      string
		available bool
		preOrder  bool
	}{
		{"in stock, not preorder", token + ` schema.org/InStock`, true, false},
		{"preorder flagged", token + ` schema.org/PreOrder`, true, true},
		{"backorder flagged", token + ` schema.org/BackOrder`, true, true},
		{"out of stock wins", token + ` schema.org/PreOrder schema.org/OutOfStock`, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newPageServer(t, tc.html)
			defer srv.Close()

			src := NewMediaMarkt(srv.Client(), nil, []string{srv.URL + "/p_" + token + ".html"})
			got, err := src.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if available := len(got) > 0; available != tc.available {
				t.Fatalf("available=%v, want %v", available, tc.available)
			}
			if tc.available && got[0].PreOrder != tc.preOrder {
				t.Errorf("preOrder=%v, want %v", got[0].PreOrder, tc.preOrder)
			}
		})
	}
}

func newPageServer(t *testing.T, html string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
}

// mmToken is the product id used by the token guard; test pages must include it.
const mmToken = "142245268"

func TestMediaMarktAvailability(t *testing.T) {
	cases := []struct {
		name      string
		html      string
		available bool
	}{
		{"in stock", `<script>{"productId":"142245268","offers":{"availability":"https://schema.org/InStock"}}</script>`, true},
		{"in stock escaped slash", "142245268 " + `{"availability":"https:\/\/schema.org\/InStock"}`, true},
		{"out of stock", `<script>{"productId":"142245268","offers":{"availability":"https://schema.org/OutOfStock"}}</script>`, false},
		{"out wins over in", `142245268 schema.org/InStock schema.org/OutOfStock`, false},
		{"no markers", `<div>142245268 but nothing structured</div>`, false},
		// Regression: soft-404 with other products' schema.org/InStock but no token.
		{"soft 404 without product token", `<div>Seite nicht gefunden</div> schema.org/InStock schema.org/InStock`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newPageServer(t, tc.html)
			defer srv.Close()

			src := NewMediaMarkt(srv.Client(), nil, []string{srv.URL + "/p_" + mmToken + ".html"})
			got, err := src.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if available := len(got) > 0; available != tc.available {
				t.Fatalf("available=%v, want %v (results=%+v)", available, tc.available, got)
			}
		})
	}
}
