package source

import (
	"net/http"

	"portasplit-monitor/internal/model"
)

// euronicsDefaultURL is the PortaSplit product page on euronics.de. When the
// product is delisted/sold out Euronics serves a soft-404 at this path, which
// the product-token guard in webCheck handles (no false positive).
const euronicsDefaultURL = "https://www.euronics.de/haus-und-haushalt/heizen-lueften-kuehlen/kuehlen/split-klimageraete/porta-split-split-klimageraet-a-4065327878899"

// EuronicsSource checks the Euronics product page for availability. Euronics has
// no public stock API and blocks plain requests, so this is a JSON-LD
// schema.org availability check that needs FlareSolverr in practice. It reports
// online availability only.
type EuronicsSource struct {
	webCheck
}

// NewEuronics builds a Euronics source. An empty url falls back to the default
// PortaSplit product page.
func NewEuronics(client *http.Client, fs *FlareSolverr, url string) *EuronicsSource {
	if url == "" {
		url = euronicsDefaultURL
	}
	return &EuronicsSource{
		webCheck: webCheck{
			name:         "euronics",
			client:       client,
			fs:           fs,
			url:          url,
			storeName:    "Euronics",
			product:      "Midea PortaSplit",
			channel:      model.ChannelOnline,
			requireToken: productToken(url),
			inStock:      schemaInStock,
			outOfStock:   schemaOutOfStock,
		},
	}
}
