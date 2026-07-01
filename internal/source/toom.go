package source

import "net/http"

// ToomSource checks toom product pages (online availability only; needs FlareSolverr).
type ToomSource struct {
	webCheck
}

// NewToom builds a toom source for the given product URLs.
func NewToom(client *http.Client, fs *FlareSolverr, urls []string) *ToomSource {
	return &ToomSource{newSchemaCheck("toom", client, fs, urls, "toom")}
}
