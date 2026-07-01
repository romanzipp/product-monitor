package source

import "net/http"

// EuronicsSource checks Euronics product pages (online availability only; needs FlareSolverr).
type EuronicsSource struct {
	webCheck
}

// NewEuronics builds a Euronics source for the given product URLs.
func NewEuronics(client *http.Client, fs *FlareSolverr, urls []string) *EuronicsSource {
	return &EuronicsSource{newSchemaCheck("euronics", client, fs, urls, "Euronics")}
}
