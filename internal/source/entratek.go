package source

import "net/http"

// EntratekSource checks entratek-shop.de product pages. Availability comes from
// reliable schema.org microdata (InStock -> "Auf Lager", OutOfStock -> "Nicht auf
// Lager"). Not anti-bot protected; fetched directly.
type EntratekSource struct {
	webCheck
}

// NewEntratek builds an entratek-shop.de source for the given product URLs.
func NewEntratek(client *http.Client, fs *FlareSolverr, urls []string) *EntratekSource {
	return &EntratekSource{newSchemaCheck("entratek", client, fs, urls, "Entratek")}
}
