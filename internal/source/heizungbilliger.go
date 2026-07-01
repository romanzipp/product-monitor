package source

import "net/http"

// HeizungBilligerSource checks heizung-billiger.de product pages (PrestaShop). An
// in-stock product emits a schema.org/InStock Offer; an out-of-stock one drops the
// Product JSON-LD entirely and shows "Aktuell nicht lieferbar". Behind Cloudflare
// (JA3 fingerprint wall), so it MUST be routed through FlareSolverr.
type HeizungBilligerSource struct {
	webCheck
}

// NewHeizungBilliger builds a heizung-billiger.de source for the given product URLs.
func NewHeizungBilliger(client *http.Client, fs *FlareSolverr, urls []string) *HeizungBilligerSource {
	wc := newSchemaCheck("heizungbilliger", client, fs, urls, "Heizung Billiger")
	wc.outOfStock = append(wc.outOfStock, "aktuell nicht lieferbar")
	return &HeizungBilligerSource{wc}
}
