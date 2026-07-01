package source

import "net/http"

// SolarHandel24Source checks the Shopify store solarhandel24.de. Availability and
// price come from the schema.org JSON-LD Offer; the shop keeps InStock even for
// pre-orders, so a "Vorbestellung" note flags the pre-order. Not anti-bot
// protected; fetched directly.
type SolarHandel24Source struct {
	webCheck
}

// NewSolarHandel24 builds a solarhandel24.de source for the given product URLs.
func NewSolarHandel24(client *http.Client, fs *FlareSolverr, urls []string) *SolarHandel24Source {
	wc := newSchemaCheck("solarhandel24", client, fs, urls, "Solarhandel24")
	wc.preOrder = append(wc.preOrder, preOrderTextMarkers...)
	wc.priceFn = shopifyPrice
	return &SolarHandel24Source{wc}
}
