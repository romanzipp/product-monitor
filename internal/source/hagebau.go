package source

import "net/http"

const hagebauDefaultURL = "https://www.hagebau.de/p/midea-klimaanlage-portasplit-anP7004600334/"

// HagebauSource checks the Hagebau product page (online; needs FlareSolverr).
// While sold out the page 404s, handled by the token guard.
type HagebauSource struct {
	webCheck
}

// NewHagebau builds a Hagebau source; an empty url uses the default page.
func NewHagebau(client *http.Client, fs *FlareSolverr, url string) *HagebauSource {
	if url == "" {
		url = hagebauDefaultURL
	}
	return &HagebauSource{newSchemaCheck("hagebau", client, fs, url, "Hagebau")}
}
