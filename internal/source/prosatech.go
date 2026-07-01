package source

import "net/http"

// ProsatechSource checks prosatech.de product pages (JTL-Shop). Availability comes
// from the schema.org JSON-LD Offer (and matching microdata). Not anti-bot
// protected; fetched directly.
type ProsatechSource struct {
	webCheck
}

// NewProsatech builds a prosatech.de source for the given product URLs.
func NewProsatech(client *http.Client, fs *FlareSolverr, urls []string) *ProsatechSource {
	return &ProsatechSource{newSchemaCheck("prosatech", client, fs, urls, "Prosatech")}
}
