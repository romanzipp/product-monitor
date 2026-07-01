package source

import "net/http"

// HornbachSource checks Hornbach product pages (online availability only; needs FlareSolverr).
type HornbachSource struct {
	webCheck
}

// NewHornbach builds a Hornbach source for the given product URLs.
func NewHornbach(client *http.Client, fs *FlareSolverr, urls []string) *HornbachSource {
	return &HornbachSource{newSchemaCheck("hornbach", client, fs, urls, "Hornbach")}
}
