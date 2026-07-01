package source

import (
	"net/http"
	"regexp"

	"product-monitor/internal/model"
)

// mediadealIDRe pulls the numeric product id from a `…/<id>/<slug>` URL path. It
// also appears on the page (e.g. "productID":19421), so it guards against soft-404s.
var mediadealIDRe = regexp.MustCompile(`/(\d+)/[^/]+/?$`)

// mediadealInStock/OutOfStock: this Shopware 5 shop emits schema.org microdata,
// but uses LimitedAvailability for NOT-orderable items ("Lieferzeit anfragen" /
// "im Zulauf"), so only InStock counts as available and LimitedAvailability is
// treated as out of stock.
var mediadealInStock = []string{"schema.org/instock"}
var mediadealOutOfStock = append([]string{"schema.org/limitedavailability", "lieferzeit anfragen"}, schemaOutOfStock...)

// MediaDealSource checks mediadeal.de product pages. Not anti-bot protected.
type MediaDealSource struct {
	webCheck
}

// NewMediaDeal builds a mediadeal.de source for the given product URLs.
func NewMediaDeal(client *http.Client, fs *FlareSolverr, urls []string) *MediaDealSource {
	return &MediaDealSource{
		webCheck: webCheck{
			name:       "mediadeal",
			client:     client,
			fs:         fs,
			urls:       urls,
			storeName:  "MediaDeal",
			product:    "Midea PortaSplit",
			channel:    model.ChannelOnline,
			tokenFn:    mediadealID,
			inStock:    mediadealInStock,
			outOfStock: mediadealOutOfStock,
			priceFn:    parsePrice,
		},
	}
}

// mediadealID extracts the numeric product id from a mediadeal.de URL path.
func mediadealID(url string) string {
	if m := mediadealIDRe.FindStringSubmatch(url); len(m) == 2 {
		return m[1]
	}
	return ""
}
