package source

import (
	"net/http"
	"regexp"

	"product-monitor/internal/model"
)

var amazonASINRe = regexp.MustCompile(`/dp/([A-Za-z0-9]{10})`)

// AmazonSource checks the Amazon product page. Amazon exposes no schema.org
// availability, so stock is inferred from the buybox add-to-cart button, which
// is only rendered when the item is buyable. Needs FlareSolverr (anti-bot), and
// reports no price (so PRICE_MAX cannot filter Amazon offers).
type AmazonSource struct {
	webCheck
}

// NewAmazon builds an Amazon source for the given product URLs.
func NewAmazon(client *http.Client, fs *FlareSolverr, urls []string) *AmazonSource {
	return &AmazonSource{
		webCheck: webCheck{
			name:      "amazon",
			client:    client,
			fs:        fs,
			urls:      urls,
			storeName: "Amazon",
			product:   "Midea PortaSplit",
			channel:   model.ChannelOnline,
			tokenFn:   amazonASIN,
			inStock:   []string{"add-to-cart-button"},
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
