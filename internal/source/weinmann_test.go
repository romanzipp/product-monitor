package source

import (
	"context"
	"testing"
)

func TestWeinmannSchanzAvailability(t *testing.T) {
	const sku = "9013479"

	// Build the entity-encoded availabilityJson blob as the live page emits it
	// (&#34; for ", \&#34; for the escaped inner quotes).
	blob := func(id int, inStock string) string {
		q := "&#34;"
		iq := `\&#34;`
		return `availabilityJson` + q + `:` + q + `{` +
			iq + `id` + iq + `:` + itoa(id) + `,` +
			iq + `message` + iq + `:` + iq + `x` + iq + `,` +
			iq + `is_in_stock` + iq + `:` + inStock + `}` + q + ` ` + sku
	}

	cases := []struct {
		name      string
		html      string
		available bool
	}{
		{"in stock id 1", blob(1, "true"), true},
		{"long delivery id 3 not available", blob(3, "true"), false},
		{"out of stock id 7", blob(7, "false"), false},
		{"soft 404 without token", `availabilityJson&#34;:&#34;{\&#34;id\&#34;:1,\&#34;is_in_stock\&#34;:true}&#34; 0000000`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newPageServer(t, tc.html)
			defer srv.Close()

			src := NewWeinmannSchanz(srv.Client(), nil, []string{srv.URL + "/produkt.html/midea-porta-split-90-134-79.html"})
			got, err := src.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if available := len(got) > 0; available != tc.available {
				t.Fatalf("available=%v, want %v (%+v)", available, tc.available, got)
			}
		})
	}
}

func itoa(i int) string {
	return string(rune('0' + i))
}
