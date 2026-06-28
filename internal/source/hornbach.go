package source

import "net/http"

const hornbachDefaultURL = "https://www.hornbach.de/p/klimasplitgeraet-midea-portasplit-12-000-btu-105-m-weiss/12356554/"

// HornbachSource checks the Hornbach product page (online; needs FlareSolverr,
// the page is JS-rendered). While sold out the page 404s, handled by the guard.
type HornbachSource struct {
	webCheck
}

// NewHornbach builds a Hornbach source; an empty url uses the default page.
func NewHornbach(client *http.Client, fs *FlareSolverr, url string) *HornbachSource {
	if url == "" {
		url = hornbachDefaultURL
	}
	return &HornbachSource{newSchemaCheck("hornbach", client, fs, url, "Hornbach")}
}
