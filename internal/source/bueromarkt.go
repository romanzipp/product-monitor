package source

import (
	"net/http"
	"regexp"

	"product-monitor/internal/model"
)

// bueromarktIDRe pulls the product id out of a `…,p-<id>.html` URL. The id also
// appears on the page as the JSON-LD sku, so it doubles as the soft-404 guard.
var bueromarktIDRe = regexp.MustCompile(`p-(\d+)\.html`)

// bueromarktPriceRe reads the gross price from the main price element's
// data-price-gross attribute (German format, e.g. "1.784,99 €").
var bueromarktPriceRe = regexp.MustCompile(`data-price-gross="([0-9][0-9.,]*)`)

// BueromarktSource checks bueromarkt-ag.de product pages. The shop exposes no
// schema.org availability, so stock is read from its status element: an in-stock
// item shows "sofort lieferbar" (class immediateDelivery), while an out-of-stock
// item shows "derzeit nicht verfügbar" (class product-order-status / available-soon
// with an email-notify block). Behind Imperva/Incapsula, so route via FlareSolverr.
type BueromarktSource struct {
	webCheck
}

// NewBueromarkt builds a bueromarkt-ag.de source for the given product URLs.
func NewBueromarkt(client *http.Client, fs *FlareSolverr, urls []string) *BueromarktSource {
	return &BueromarktSource{
		webCheck: webCheck{
			name:       "bueromarkt",
			client:     client,
			fs:         fs,
			urls:       urls,
			storeName:  "Büromarkt AG",
			product:    "Midea PortaSplit",
			channel:    model.ChannelOnline,
			tokenFn:    bueromarktID,
			inStock:    []string{"immediatedelivery", "sofort lieferbar"},
			outOfStock: []string{"derzeit nicht verf", "notify-product-"},
			priceFn:    bueromarktPrice,
		},
	}
}

// bueromarktID extracts the product id from a `,p-<id>.html` URL.
func bueromarktID(url string) string {
	if m := bueromarktIDRe.FindStringSubmatch(url); len(m) == 2 {
		return m[1]
	}
	return ""
}

// bueromarktPrice extracts the gross price from a (lowercased) page, or nil.
func bueromarktPrice(html string) *float64 {
	if m := bueromarktPriceRe.FindStringSubmatch(html); m != nil {
		return parseAmount(m[1])
	}
	return nil
}
