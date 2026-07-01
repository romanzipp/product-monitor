package source

import "net/http"

// Solario24Source checks solario24.com product pages (WooCommerce). Availability
// comes from the schema.org JSON-LD Offer; the shop is not anti-bot protected, so
// it is fetched directly (no FlareSolverr needed).
type Solario24Source struct {
	webCheck
}

// NewSolario24 builds a solario24.com source for the given product URLs.
func NewSolario24(client *http.Client, fs *FlareSolverr, urls []string) *Solario24Source {
	wc := newSchemaCheck("solario24", client, fs, urls, "Solario24")
	// WooCommerce keeps the Offer as InStock for backorderable items, so flag the
	// "Vorbestellung" / "Aktuell nicht lieferbar" note as a pre-order.
	wc.preOrder = append(wc.preOrder, "vorbestellung", "aktuell nicht lieferbar")
	return &Solario24Source{wc}
}
