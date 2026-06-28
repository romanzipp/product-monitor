package source

import (
	"net/http"

	"portasplit-monitor/internal/model"
)

// euronicsDefaultURL is the PortaSplit product page on euronics.de. When the
// product is delisted Euronics serves a soft-404 here (handled by the token guard).
const euronicsDefaultURL = "https://www.euronics.de/haus-und-haushalt/heizen-lueften-kuehlen/kuehlen/split-klimageraete/porta-split-split-klimageraet-a-4065327878899"

// EuronicsSource checks the Euronics product page (online availability only;
// needs FlareSolverr in practice).
type EuronicsSource struct {
	webCheck
}

// NewEuronics builds a Euronics source; an empty url uses the default page.
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
