package source

import "net/http"

// KlimafySource checks klimafy.de product pages (Shopware 6). The reliable signal
// is the schema.org microdata link (InStock/OutOfStock); the JSON-LD Offer adds
// SoldOut and PreOrder (escaped slashes, normalised by the shared engine). Its
// JSON-LD LimitedAvailability appears only on genuinely in-stock items, so the
// standard detection is safe here. Not anti-bot protected; fetched directly.
type KlimafySource struct {
	webCheck
}

// NewKlimafy builds a klimafy.de source for the given product URLs.
func NewKlimafy(client *http.Client, fs *FlareSolverr, urls []string) *KlimafySource {
	return &KlimafySource{newSchemaCheck("klimafy", client, fs, urls, "Klimafy")}
}
