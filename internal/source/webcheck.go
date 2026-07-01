package source

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"product-monitor/internal/model"
)

// JSON-LD schema.org availability tokens. The structured Offer availability is
// far more reliable than visible text, which is polluted by recommended products.
var schemaInStock = []string{"schema.org/instock", "schema.org/limitedavailability", "schema.org/preorder"}
var schemaOutOfStock = []string{"schema.org/outofstock", "schema.org/soldout", "schema.org/discontinued"}

var digitRunRe = regexp.MustCompile(`[0-9]{6,}`)

// newSchemaCheck builds a webCheck for a retailer that exposes stock via the
// standard schema.org JSON-LD Offer availability across one or more product URLs.
func newSchemaCheck(name string, client *http.Client, fs *FlareSolverr, urls []string, storeName string) webCheck {
	return webCheck{
		name:       name,
		client:     client,
		fs:         fs,
		urls:       urls,
		storeName:  storeName,
		product:    "Midea PortaSplit",
		channel:    model.ChannelOnline,
		tokenFn:    productToken,
		inStock:    schemaInStock,
		outOfStock: schemaOutOfStock,
		priceFn:    parsePrice,
	}
}

// schema.org Offer price, as microdata (<meta itemprop="price" content="…">) or
// JSON-LD ("price": 799.00). The JSON form requires a digit after the colon to
// skip label strings like "price":"Preis".
var priceMicroRe = regexp.MustCompile(`itemprop="price"[^>]*content="([0-9](?:[0-9.,]*[0-9])?)"`)
var priceJSONRe = regexp.MustCompile(`"price"\s*:\s*"?([0-9](?:[0-9.,]*[0-9])?)`)

// parsePrice extracts the schema.org price from a (lowercased) page, or nil.
func parsePrice(html string) *float64 {
	if m := priceMicroRe.FindStringSubmatch(html); m != nil {
		return parseAmount(m[1])
	}
	if m := priceJSONRe.FindStringSubmatch(html); m != nil {
		return parseAmount(m[1])
	}
	return nil
}

// parseAmount parses a price string, normalising German formats like "2.949,99"
// and "799,00", and returns nil for a non-positive or invalid value.
func parseAmount(raw string) *float64 {
	if raw == "" {
		return nil
	}
	if strings.Contains(raw, ".") && strings.Contains(raw, ",") {
		raw = strings.ReplaceAll(raw, ".", "")
		raw = strings.ReplaceAll(raw, ",", ".")
	} else if strings.Contains(raw, ",") {
		raw = strings.ReplaceAll(raw, ",", ".")
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || f <= 0 {
		return nil
	}
	return &f
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

// webCheck is the shared engine for retailers with no availability API. It checks
// one or more product pages (via FlareSolverr when configured) and decides stock
// from the schema.org markers in the embedded JSON-LD. Vendor sources embed it.
type webCheck struct {
	name       string
	client     *http.Client
	fs         *FlareSolverr
	urls       []string
	storeName  string
	product    string
	channel    model.Channel
	tokenFn    func(string) string // required page token per URL (empty = no guard)
	inStock    []string
	outOfStock []string
	priceFn    func(string) *float64 // extracts price from the page (nil = no price)
}

func (s *webCheck) Name() string {
	return s.name
}

func (s *webCheck) Check(ctx context.Context) ([]model.Availability, error) {
	out := make([]model.Availability, 0, len(s.urls))
	var errs []error
	for _, url := range s.urls {
		a, err := s.checkOne(ctx, url)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if a != nil {
			out = append(out, *a)
		}
	}
	// Surface an error only when every URL failed; partial success still returns
	// the results it found.
	if len(out) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// checkOne fetches a single product page and returns an availability if in stock.
func (s *webCheck) checkOne(ctx context.Context, url string) (*model.Availability, error) {
	headers := map[string]string{"User-Agent": browserUserAgent, "Accept": "text/html,application/xhtml+xml", "Accept-Language": "de-DE,de;q=0.9"}
	body, err := getBody(ctx, s.client, s.fs, url, headers)
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	html := strings.ToLower(string(body))

	// Soft-404 guard: delisted products return a 200 page full of OTHER in-stock
	// items. If this product's id/EAN is absent, it is not really available.
	if s.tokenFn != nil {
		if token := s.tokenFn(url); token != "" && !strings.Contains(html, strings.ToLower(token)) {
			return nil, nil
		}
	}

	for _, marker := range s.outOfStock {
		if strings.Contains(html, strings.ToLower(marker)) {
			return nil, nil
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
		return nil, nil
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
		URL:         url,
		Location:    location,
		Channel:     s.channel,
		Key:         s.name + ":" + url,
	}
	if s.priceFn != nil {
		a.Price = s.priceFn(html)
	}
	return &a, nil
}
