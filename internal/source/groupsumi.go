package source

import "net/http"

// GroupSumiSource checks groupsumi.de product pages. Availability and price come
// from the schema.org JSON-LD Offer. Not anti-bot protected; fetched directly.
type GroupSumiSource struct {
	webCheck
}

// NewGroupSumi builds a groupsumi.de source for the given product URLs.
func NewGroupSumi(client *http.Client, fs *FlareSolverr, urls []string) *GroupSumiSource {
	return &GroupSumiSource{newSchemaCheck("groupsumi", client, fs, urls, "Group Sumi")}
}
