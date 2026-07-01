package source

import "net/http"

// preOrderTextMarkers are German pre-order phrases used by Shopify shops that keep
// the schema.org Offer as InStock while the item is actually a pre-order. The
// phrases include a trailing separator so they don't match the bare theme label
// "Vorbestellung" that Shopify themes ship in their variant-string scaffolding.
var preOrderTextMarkers = []string{"vorbestellung -", "vorbestellung mög", "pre-order"}

// TadoSource checks the Shopify store shop.tado.com. Availability and price come
// from the schema.org JSON-LD Offer; a "Vorbestellung" note flags a pre-order.
// Not anti-bot protected; fetched directly.
type TadoSource struct {
	webCheck
}

// NewTado builds a shop.tado.com source for the given product URLs.
func NewTado(client *http.Client, fs *FlareSolverr, urls []string) *TadoSource {
	wc := newSchemaCheck("tado", client, fs, urls, "tado")
	wc.preOrder = append(wc.preOrder, preOrderTextMarkers...)
	wc.priceFn = shopifyPrice
	return &TadoSource{wc}
}
