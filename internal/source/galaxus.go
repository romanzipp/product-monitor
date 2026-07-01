package source

import "net/http"

// GalaxusSource checks galaxus.de product pages. Galaxus sits behind Akamai Bot
// Manager with an active CAPTCHA tier, so it MUST be routed through FlareSolverr
// (and even that may hit the CAPTCHA and need a residential proxy). Availability
// is read from the standard schema.org JSON-LD Offer.
type GalaxusSource struct {
	webCheck
}

// NewGalaxus builds a galaxus.de source for the given product URLs.
func NewGalaxus(client *http.Client, fs *FlareSolverr, urls []string) *GalaxusSource {
	return &GalaxusSource{newSchemaCheck("galaxus", client, fs, urls, "Galaxus")}
}
