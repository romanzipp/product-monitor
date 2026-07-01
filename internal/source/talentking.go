package source

import "net/http"

// TalentKingSource checks the Shopify store talent-king.de. Availability and price
// come from the schema.org JSON-LD Offer (http:// prefix, matched case-insensitively);
// a "Vorbestellung" note flags a pre-order. Not anti-bot protected; fetched directly.
type TalentKingSource struct {
	webCheck
}

// NewTalentKing builds a talent-king.de source for the given product URLs.
func NewTalentKing(client *http.Client, fs *FlareSolverr, urls []string) *TalentKingSource {
	wc := newSchemaCheck("talentking", client, fs, urls, "Talent King")
	wc.preOrder = append(wc.preOrder, preOrderTextMarkers...)
	return &TalentKingSource{wc}
}
