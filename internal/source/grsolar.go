package source

import "net/http"

// GrSolarSource checks gr-solar.de product pages (WooCommerce). Availability and
// price come from the reliable schema.org JSON-LD Offer (InStock -> "Vorrätig",
// OutOfStock -> "Nicht vorrätig"). Not anti-bot protected; fetched directly.
type GrSolarSource struct {
	webCheck
}

// NewGrSolar builds a gr-solar.de source for the given product URLs.
func NewGrSolar(client *http.Client, fs *FlareSolverr, urls []string) *GrSolarSource {
	return &GrSolarSource{newSchemaCheck("grsolar", client, fs, urls, "GR Solar")}
}
