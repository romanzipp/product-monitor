package source

import "net/http"

// HagebauSource checks Hagebau product pages (online availability only; needs FlareSolverr).
type HagebauSource struct {
	webCheck
}

// NewHagebau builds a Hagebau source for the given product URLs.
func NewHagebau(client *http.Client, fs *FlareSolverr, urls []string) *HagebauSource {
	return &HagebauSource{newSchemaCheck("hagebau", client, fs, urls, "Hagebau")}
}
