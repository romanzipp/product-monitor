package source

import "net/http"

// mediaMarktDefaultURL is the PortaSplit product page (grey, 42 m² variant).
const mediaMarktDefaultURL = "https://www.mediamarkt.de/de/product/_midea-porta-split-klimaanlage-grau-max-raumgrosse-42-m-eek-a-142245268.html"

// MediaMarktSource checks the MediaMarkt product page (online availability only;
// needs FlareSolverr in practice).
type MediaMarktSource struct {
	webCheck
}

// NewMediaMarkt builds a MediaMarkt source; an empty url uses the default page.
func NewMediaMarkt(client *http.Client, fs *FlareSolverr, url string) *MediaMarktSource {
	if url == "" {
		url = mediaMarktDefaultURL
	}
	return &MediaMarktSource{newSchemaCheck("mediamarkt", client, fs, url, "MediaMarkt")}
}
