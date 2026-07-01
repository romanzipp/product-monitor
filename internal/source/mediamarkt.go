package source

import "net/http"

// MediaMarktSource checks MediaMarkt product pages (online availability only; needs FlareSolverr).
type MediaMarktSource struct {
	webCheck
}

// NewMediaMarkt builds a MediaMarkt source for the given product URLs.
func NewMediaMarkt(client *http.Client, fs *FlareSolverr, urls []string) *MediaMarktSource {
	return &MediaMarktSource{newSchemaCheck("mediamarkt", client, fs, urls, "MediaMarkt")}
}
