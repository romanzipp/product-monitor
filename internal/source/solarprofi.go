package source

import (
	"net/http"
	"regexp"

	"product-monitor/internal/model"
)

var solarprofiPathRe = regexp.MustCompile(`/(\d{3,})(?:/|$)`)

// solarprofiToken returns the numeric path-segment id from a shop URL, e.g.
// ".../haustechnik/7212/..." -> "7212". The generic productToken misses it
// because the id is shorter than an EAN.
func solarprofiToken(url string) string {
	if m := solarprofiPathRe.FindStringSubmatch(url); len(m) == 2 {
		return m[1]
	}
	return ""
}

// SolarProfiSource checks solarprofi-24.de product pages. The shop is not
// anti-bot protected and exposes schema.org microdata availability, so it is
// fetched directly (no FlareSolverr needed).
type SolarProfiSource struct {
	webCheck
}

// NewSolarProfi builds a solarprofi-24 source for the given product URLs.
func NewSolarProfi(client *http.Client, fs *FlareSolverr, urls []string) *SolarProfiSource {
	return &SolarProfiSource{
		webCheck: webCheck{
			name:       "solarprofi",
			client:     client,
			fs:         fs,
			urls:       urls,
			storeName:  "Solarprofi 24",
			product:    "Midea PortaSplit",
			channel:    model.ChannelOnline,
			tokenFn:    solarprofiToken,
			inStock:    schemaInStock,
			outOfStock: schemaOutOfStock,
			priceFn:    parsePrice,
		},
	}
}
