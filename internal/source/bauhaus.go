package source

import "net/http"

const bauhausDefaultURL = "https://www.bauhaus.info/p/31934233"

// BauhausSource checks the Bauhaus product page (online; needs FlareSolverr,
// the page 403s plain requests).
type BauhausSource struct {
	webCheck
}

// NewBauhaus builds a Bauhaus source; an empty url uses the default page.
func NewBauhaus(client *http.Client, fs *FlareSolverr, url string) *BauhausSource {
	if url == "" {
		url = bauhausDefaultURL
	}
	return &BauhausSource{newSchemaCheck("bauhaus", client, fs, url, "Bauhaus")}
}
