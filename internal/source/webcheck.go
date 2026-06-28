package source

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"portasplit-monitor/internal/model"
)

// WebCheckSource monitors a single product page that exposes no usable JSON
// availability API (e.g. MediaMarkt, Euronics). It fetches the page HTML
// (through FlareSolverr when configured, to clear anti-bot challenges) and
// decides stock by looking for marker strings:
//
//   - if any outOfStock marker is present, the item is treated as unavailable;
//   - otherwise, if any inStock marker is present, it is treated as available.
//
// Markers are matched case-insensitively. This is inherently more brittle than
// a structured feed and may need tuning if a retailer changes its markup.
type WebCheckSource struct {
	name       string
	client     *http.Client
	fs         *FlareSolverr
	url        string
	storeName  string
	product    string
	channel    model.Channel
	inStock    []string
	outOfStock []string
}

// NewWebCheck builds a page-scraping source. name is the logical source id,
// storeName the display name, and inStock/outOfStock the marker strings used to
// decide availability.
func NewWebCheck(name string, client *http.Client, fs *FlareSolverr, url, storeName, product string,
	channel model.Channel, inStock, outOfStock []string) *WebCheckSource {
	return &WebCheckSource{
		name:       name,
		client:     client,
		fs:         fs,
		url:        url,
		storeName:  storeName,
		product:    product,
		channel:    channel,
		inStock:    inStock,
		outOfStock: outOfStock,
	}
}

func (s *WebCheckSource) Name() string { return s.name }

func (s *WebCheckSource) Check(ctx context.Context) ([]model.Availability, error) {
	body, err := getBody(ctx, s.client, s.fs, s.url, map[string]string{
		"User-Agent":      browserUserAgent,
		"Accept":          "text/html,application/xhtml+xml",
		"Accept-Language": "de-DE,de;q=0.9",
	})
	if err != nil {
		return nil, err
	}

	html := strings.ToLower(string(body))

	for _, m := range s.outOfStock {
		if strings.Contains(html, strings.ToLower(m)) {
			return []model.Availability{}, nil
		}
	}

	available := false
	for _, m := range s.inStock {
		if strings.Contains(html, strings.ToLower(m)) {
			available = true
			break
		}
	}
	if !available {
		return []model.Availability{}, nil
	}

	loc := "Online"
	if s.channel == model.ChannelInStore {
		loc = s.storeName
	}
	return []model.Availability{{
		Source:      s.name,
		StoreName:   s.storeName,
		ProductName: s.product,
		Stock:       1, // page-level check: presence implies at least one available
		URL:         s.url,
		Location:    loc,
		Channel:     s.channel,
		Key:         fmt.Sprintf("%s:%s", s.name, s.url),
	}}, nil
}
