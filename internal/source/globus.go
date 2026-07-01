package source

import "net/http"

// GlobusSource checks Globus Baumarkt product pages (online availability only; needs FlareSolverr).
type GlobusSource struct {
	webCheck
}

// NewGlobus builds a Globus Baumarkt source for the given product URLs.
func NewGlobus(client *http.Client, fs *FlareSolverr, urls []string) *GlobusSource {
	return &GlobusSource{newSchemaCheck("globus", client, fs, urls, "Globus Baumarkt")}
}
