package source

import "net/http"

// SchwabKlimaSource checks schwab-klima.de product pages (Wix). The availability
// is embedded server-side in the page as `"Availability":"https://schema.org/…"`
// (non-standard capitalisation, matched case-insensitively by the schema check).
// Not anti-bot protected; fetched directly.
type SchwabKlimaSource struct {
	webCheck
}

// NewSchwabKlima builds a schwab-klima.de source for the given product URLs.
func NewSchwabKlima(client *http.Client, fs *FlareSolverr, urls []string) *SchwabKlimaSource {
	return &SchwabKlimaSource{newSchemaCheck("schwabklima", client, fs, urls, "Schwab Klima")}
}
