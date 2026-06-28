package source

import (
	"net/http"

	"portasplit-monitor/internal/model"
)

// globusDefaultURL is the PortaSplit product page on globus-baumarkt.de. While
// sold out Globus serves a 404 here (handled by the token guard).
const globusDefaultURL = "https://www.globus-baumarkt.de/p/midea-portasplit-mobile-split-klimaanlage-12000-btu-heiz-kuehlfunktion-0694600235/"

// GlobusSource checks the Globus Baumarkt product page (online availability only;
// physical-store stock is covered by braucheklima).
type GlobusSource struct {
	webCheck
}

// NewGlobus builds a Globus source; an empty url uses the default page.
func NewGlobus(client *http.Client, fs *FlareSolverr, url string) *GlobusSource {
	if url == "" {
		url = globusDefaultURL
	}
	return &GlobusSource{
		webCheck: webCheck{
			name:         "globus",
			client:       client,
			fs:           fs,
			url:          url,
			storeName:    "Globus Baumarkt",
			product:      "Midea PortaSplit",
			channel:      model.ChannelOnline,
			requireToken: productToken(url),
			inStock:      schemaInStock,
			outOfStock:   schemaOutOfStock,
		},
	}
}
