package source

import (
	"net/http"

	"portasplit-monitor/internal/model"
)

// mediaMarktDefaultURL is the PortaSplit product page (grey, 42 m² variant).
const mediaMarktDefaultURL = "https://www.mediamarkt.de/de/product/_midea-porta-split-klimaanlage-grau-max-raumgrosse-42-m-eek-a-142245268.html"

// MediaMarktSource checks the MediaMarkt product page for availability.
// MediaMarkt exposes no public stock API and protects its pages with anti-bot,
// so this is a JSON-LD schema.org availability check that needs FlareSolverr in
// practice. It reports online availability only (not per-market in-store stock).
type MediaMarktSource struct {
	webCheck
}

// NewMediaMarkt builds a MediaMarkt source. An empty url falls back to the
// default PortaSplit product page.
func NewMediaMarkt(client *http.Client, fs *FlareSolverr, url string) *MediaMarktSource {
	if url == "" {
		url = mediaMarktDefaultURL
	}
	return &MediaMarktSource{
		webCheck: webCheck{
			name:         "mediamarkt",
			client:       client,
			fs:           fs,
			url:          url,
			storeName:    "MediaMarkt",
			product:      "Midea PortaSplit",
			channel:      model.ChannelOnline,
			requireToken: productToken(url),
			inStock:      schemaInStock,
			outOfStock:   schemaOutOfStock,
		},
	}
}
