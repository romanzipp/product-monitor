package source

import "net/http"

// BauhausSource checks Bauhaus product pages (online availability only; needs FlareSolverr).
type BauhausSource struct {
	webCheck
}

// NewBauhaus builds a Bauhaus source for the given product URLs.
func NewBauhaus(client *http.Client, fs *FlareSolverr, urls []string) *BauhausSource {
	return &BauhausSource{newSchemaCheck("bauhaus", client, fs, urls, "Bauhaus")}
}
