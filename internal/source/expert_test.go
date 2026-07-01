package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExpertAvailability(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		available bool
		preOrder  bool
		price     float64
	}{
		{
			"in stock",
			`{"price":{"onlineButtonAction":"ORDER","onlineStock":1,"onlineAvailability":"AVAILABLE","bruttoPrice":14.99}}`,
			true, false, 14.99,
		},
		{
			"pre-order",
			`{"price":{"onlineButtonAction":"ORDER_IN_ADVANCE","onlineStock":0,"onlineAvailability":"PREORDER","bruttoPrice":999.0}}`,
			true, true, 999,
		},
		{
			"sold out",
			`{"price":{"onlineButtonAction":"NONE","onlineStock":0,"storeAvailability":"SOLD_OUT"}}`,
			false, false, 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			src := NewExpert(srv.Client(), []string{"https://www.expert.de/shop/x/32750011559-portasplit.html"}, "e_2879130")
			src.baseURL = srv.URL

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
				if got[0].Price == nil || *got[0].Price != tc.price {
					t.Errorf("price=%v, want %v", got[0].Price, tc.price)
				}
			}
		})
	}
}
