package source

import "net/http"

// EntratekSource checks entratek-shop.de product pages. Availability comes from
// reliable schema.org microdata (InStock -> "Auf Lager", OutOfStock -> "Nicht auf
// Lager"). Incoming stock ("Im Zulauf", e.g. "Lieferzeit: 35 Tag(e)") is still
// marked InStock, so it is flagged as a pre-order. Not anti-bot protected.
type EntratekSource struct {
	webCheck
}

// NewEntratek builds an entratek-shop.de source for the given product URLs.
func NewEntratek(client *http.Client, fs *FlareSolverr, urls []string) *EntratekSource {
	wc := newSchemaCheck("entratek", client, fs, urls, "Entratek")
	wc.preOrder = append(wc.preOrder, "im zulauf")
	return &EntratekSource{wc}
}
