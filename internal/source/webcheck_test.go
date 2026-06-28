package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

			src := NewMediaMarkt(srv.Client(), nil, srv.URL+"/p_"+mmToken+".html")
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
