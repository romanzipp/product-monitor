package source

import (
	"net/http"
	"regexp"

	"product-monitor/internal/model"
)

var amazonASINRe = regexp.MustCompile(`/dp/([A-Za-z0-9]{10})`)

// amazonBuyboxPriceRe pulls the price out of the first a-offscreen span (the
// machine-readable amount), which on a product page is the buybox price. It runs
// only for an in-stock result, so it isn't reached for OOS/bot pages.
var amazonBuyboxPriceRe = regexp.MustCompile(`a-offscreen">\s*([0-9][0-9.,]*)`)

// AmazonSource checks the Amazon product page. Amazon exposes no schema.org
// availability, so stock is decided by the buybox: the real add-to-cart button
// (id="add-to-cart-button") is only rendered when buyable, and id="outOfStock"
// appears when it is not. Needs FlareSolverr (anti-bot). The ASIN must be present
// on the page (token guard), which also rejects bot/captcha interstitials.
type AmazonSource struct {
	webCheck
}

// NewAmazon builds an Amazon source for the given product URLs.
func NewAmazon(client *http.Client, fs *FlareSolverr, urls []string) *AmazonSource {
	return &AmazonSource{
		webCheck: webCheck{
			name:       "amazon",
			client:     client,
			fs:         fs,
			urls:       urls,
			storeName:  "Amazon",
			product:    "Midea PortaSplit",
			channel:    model.ChannelOnline,
			tokenFn:    amazonASIN,
			inStock:    []string{`id="add-to-cart-button"`},
			outOfStock: []string{`id="outofstock"`},
			priceFn:    amazonBuyboxPrice,
		},
	}
}

// amazonASIN extracts the 10-char ASIN from a /dp/<ASIN> URL.
func amazonASIN(url string) string {
	if m := amazonASINRe.FindStringSubmatch(url); len(m) == 2 {
		return m[1]
	}
	return ""
}

// amazonBuyboxPrice extracts the buybox price from a (lowercased) page, or nil.
func amazonBuyboxPrice(html string) *float64 {
	if m := amazonBuyboxPriceRe.FindStringSubmatch(html); m != nil {
		return parseAmount(m[1])
	}
	return nil
}
