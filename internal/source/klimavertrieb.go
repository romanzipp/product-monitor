package source

import "net/http"

// KlimaVertriebSource checks klima-vertrieb.de product pages (Shopware 6).
// Availability comes from the schema.org microdata Offer. Not anti-bot protected;
// fetched directly.
type KlimaVertriebSource struct {
	webCheck
}

// NewKlimaVertrieb builds a klima-vertrieb.de source for the given product URLs.
func NewKlimaVertrieb(client *http.Client, fs *FlareSolverr, urls []string) *KlimaVertriebSource {
	return &KlimaVertriebSource{newSchemaCheck("klimavertrieb", client, fs, urls, "Klima-Vertrieb")}
}
