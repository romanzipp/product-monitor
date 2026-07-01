package source

import "net/http"

// SelfioSource checks selfio.de product pages (Shopware 6). Availability comes
// from the schema.org JSON-LD Offer (InStock/LimitedAvailability vs
// OutOfStock/SoldOut). Not anti-bot protected; fetched directly.
type SelfioSource struct {
	webCheck
}

// NewSelfio builds a selfio.de source for the given product URLs.
func NewSelfio(client *http.Client, fs *FlareSolverr, urls []string) *SelfioSource {
	return &SelfioSource{newSchemaCheck("selfio", client, fs, urls, "Selfio")}
}
