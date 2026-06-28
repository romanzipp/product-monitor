package source

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"portasplit-monitor/internal/model"
)

// schemaInStock and schemaOutOfStock are JSON-LD schema.org availability tokens.
// Matching the structured Offer availability is far more reliable than visible
// page text, which on these retailers is polluted with recommended-product
// blurbs that carry their own "in stock"/"sold out" wording.
var schemaInStock = []string{"schema.org/instock", "schema.org/limitedavailability", "schema.org/preorder"}
var schemaOutOfStock = []string{"schema.org/outofstock", "schema.org/soldout", "schema.org/discontinued"}

var digitRunRe = regexp.MustCompile(`[0-9]{6,}`)

// productToken returns the longest run of digits in a product URL (its article
// id or EAN). It is used to confirm a fetched page really is that product's
// page before trusting availability markers.
func productToken(url string) string {
	longest := ""
	for _, m := range digitRunRe.FindAllString(url, -1) {
		if len(m) > len(longest) {
			longest = m
		}
	}
	return longest
}

// webCheck is the shared engine for retailers that expose no usable availability
// API (MediaMarkt, Euronics). It fetches the product page (through FlareSolverr
// when configured, to clear anti-bot challenges) and decides stock from the
// schema.org availability markers in the page's embedded JSON-LD.
//
// Per-vendor sources embed this and supply their own identity and URL.
type webCheck struct {
	name         string
	client       *http.Client
	fs           *FlareSolverr
	url          string
	storeName    string
	product      string
	channel      model.Channel
	requireToken string // product id/EAN that must appear on the page
	inStock      []string
	outOfStock   []string
}

func (s *webCheck) Name() string {
	return s.name
}

func (s *webCheck) Check(ctx context.Context) ([]model.Availability, error) {
	headers := map[string]string{"User-Agent": browserUserAgent, "Accept": "text/html,application/xhtml+xml", "Accept-Language": "de-DE,de;q=0.9"}
	body, err := getBody(ctx, s.client, s.fs, s.url, headers)
	if err != nil {
		return nil, err
	}

	html := strings.ToLower(string(body))

	// Guard against redirects and soft-404 pages. When the product is delisted
	// these retailers serve a 200 "not found" page full of OTHER in-stock items
	// (each carrying schema.org/InStock), which would otherwise be a false
	// positive. If the product's own id/EAN is absent, treat it as unavailable.
	if s.requireToken != "" && !strings.Contains(html, strings.ToLower(s.requireToken)) {
		return []model.Availability{}, nil
	}

	for _, marker := range s.outOfStock {
		if strings.Contains(html, strings.ToLower(marker)) {
			return []model.Availability{}, nil
		}
	}

	available := false
	for _, marker := range s.inStock {
		if strings.Contains(html, strings.ToLower(marker)) {
			available = true
			break
		}
	}
	if !available {
		return []model.Availability{}, nil
	}

	location := "Online"
	if s.channel == model.ChannelInStore {
		location = s.storeName
	}

	a := model.Availability{
		Source:      s.name,
		StoreName:   s.storeName,
		ProductName: s.product,
		Stock:       1, // page-level check: presence implies at least one available
		URL:         s.url,
		Location:    location,
		Channel:     s.channel,
		Key:         s.name + ":" + s.url,
	}
	return []model.Availability{a}, nil
}
