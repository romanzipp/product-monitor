package source

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"portasplit-monitor/internal/model"
)

// JSON-LD schema.org availability tokens. The structured Offer availability is
// far more reliable than visible text, which is polluted by recommended products.
var schemaInStock = []string{"schema.org/instock", "schema.org/limitedavailability", "schema.org/preorder"}
var schemaOutOfStock = []string{"schema.org/outofstock", "schema.org/soldout", "schema.org/discontinued"}

var digitRunRe = regexp.MustCompile(`[0-9]{6,}`)

// newSchemaCheck builds a webCheck for a retailer that exposes stock via the
// standard schema.org JSON-LD Offer availability.
func newSchemaCheck(name string, client *http.Client, fs *FlareSolverr, url, storeName string) webCheck {
	return webCheck{
		name:         name,
		client:       client,
		fs:           fs,
		url:          url,
		storeName:    storeName,
		product:      "Midea PortaSplit",
		channel:      model.ChannelOnline,
		requireToken: productToken(url),
		inStock:      schemaInStock,
		outOfStock:   schemaOutOfStock,
	}
}

// productToken returns the longest run of digits in a URL (its article id/EAN).
func productToken(url string) string {
	longest := ""
	for _, m := range digitRunRe.FindAllString(url, -1) {
		if len(m) > len(longest) {
			longest = m
		}
	}
	return longest
}

// webCheck is the shared engine for retailers with no availability API. It
// fetches the product page (via FlareSolverr when configured) and decides stock
// from the schema.org markers in the embedded JSON-LD. Vendor sources embed it.
type webCheck struct {
	name         string
	client       *http.Client
	fs           *FlareSolverr
	url          string
	storeName    string
	product      string
	channel      model.Channel
	requireToken string // must appear on the page to confirm it is the right one
	inStock      []string
	outOfStock   []string
}

func (s *webCheck) Name() string {
	return s.name
}

func (s *webCheck) Check(ctx context.Context) ([]model.Availability, error) {
	headers := map[string]string{"User-Agent": browserUserAgent, "Accept": "text/html,application/xhtml+xml", "Accept-Language": "de-DE,de;q=0.9"}
	body, err := getBody(ctx, s.client, s.fs, s.url, headers)
	if errors.Is(err, errNotFound) {
		return []model.Availability{}, nil
	}
	if err != nil {
		return nil, err
	}

	html := strings.ToLower(string(body))

	// Soft-404 guard: delisted products return a 200 page full of OTHER in-stock
	// items. If this product's id/EAN is absent, it is not really available.
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
		Stock:       1, // page check: presence implies availability
		URL:         s.url,
		Location:    location,
		Channel:     s.channel,
		Key:         s.name + ":" + s.url,
	}
	return []model.Availability{a}, nil
}
