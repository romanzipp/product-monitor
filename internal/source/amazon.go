package source

import (
	"net/http"
	"regexp"

	"product-monitor/internal/model"
)

const amazonDefaultURL = "https://www.amazon.de/dp/B0D3PP64JS"

var amazonASINRe = regexp.MustCompile(`/dp/([A-Za-z0-9]{10})`)

// AmazonSource checks the Amazon product page. Amazon exposes no schema.org
// availability, so stock is inferred from the buybox add-to-cart button, which
// is only rendered when the item is buyable. Needs FlareSolverr (anti-bot), and
// reports no price (so PRICE_MAX cannot filter Amazon offers).
type AmazonSource struct {
	webCheck
}

// NewAmazon builds an Amazon source; an empty url uses the default page.
func NewAmazon(client *http.Client, fs *FlareSolverr, url string) *AmazonSource {
	if url == "" {
		url = amazonDefaultURL
	}
	return &AmazonSource{
		webCheck: webCheck{
			name:         "amazon",
			client:       client,
			fs:           fs,
			url:          url,
			storeName:    "Amazon",
			product:      "Midea PortaSplit",
			channel:      model.ChannelOnline,
			requireToken: amazonASIN(url),
			inStock:      []string{"add-to-cart-button"},
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
