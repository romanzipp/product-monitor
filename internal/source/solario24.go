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
	return &Solario24Source{newSchemaCheck("solario24", client, fs, urls, "Solario24")}
}
