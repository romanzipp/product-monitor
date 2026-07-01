package source

import "net/http"

// TecedoSource checks tecedo.de product pages. Availability comes from reliable
// schema.org microdata (InStock -> "Auf Lager", OutOfStock -> "Kein Liefertermin
// bekannt."). The page is served as ISO-8859-15, but all markers and the price are
// ASCII, so byte-wise matching works. Not anti-bot protected; fetched directly.
type TecedoSource struct {
	webCheck
}

// NewTecedo builds a tecedo.de source for the given product URLs.
func NewTecedo(client *http.Client, fs *FlareSolverr, urls []string) *TecedoSource {
	return &TecedoSource{newSchemaCheck("tecedo", client, fs, urls, "Tecedo")}
}
