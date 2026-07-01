package source

import "net/http"

// EvolarShopSource checks evolarshop.de product pages (Magento). Availability
// comes from the schema.org JSON-LD Offer (which uses the http:// scheme prefix,
// matched case-insensitively). Not anti-bot protected; fetched directly.
type EvolarShopSource struct {
	webCheck
}

// NewEvolarShop builds an evolarshop.de source for the given product URLs.
func NewEvolarShop(client *http.Client, fs *FlareSolverr, urls []string) *EvolarShopSource {
	return &EvolarShopSource{newSchemaCheck("evolarshop", client, fs, urls, "Evolar Shop")}
}
