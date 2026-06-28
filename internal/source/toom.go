package source

import "net/http"

const toomDefaultURL = "https://toom.de/p/mobiles-klimageraet-portasplit-12000-btuh/9350668"

// ToomSource checks the toom product page (online; needs FlareSolverr, the
// availability is JS-rendered and absent from the raw HTML).
type ToomSource struct {
	webCheck
}

// NewToom builds a toom source; an empty url uses the default page.
func NewToom(client *http.Client, fs *FlareSolverr, url string) *ToomSource {
	if url == "" {
		url = toomDefaultURL
	}
	return &ToomSource{newSchemaCheck("toom", client, fs, url, "toom")}
}
