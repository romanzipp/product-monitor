package source

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"product-monitor/internal/model"
)

// bobselektro.de (Gambio) exposes no schema.org data. Stock is a coloured delivery
// badge: green/yellow = orderable (short/longer lead time), red = "nicht mehr
// lieferbar". Pages carry several delivery badges (cross-sell tiles), so only the
// FIRST one (the main product, rendered above the price) is authoritative.
var bobDeliveryRe = regexp.MustCompile(`class="deliverytime deliverytime-(red|green|yellow)"`)

// bobPriceRe reads the first price block's German-formatted amount (e.g. "1.849,00").
var bobPriceRe = regexp.MustCompile(`class="price"[\s\S]{0,200}?([0-9]{1,3}(?:\.[0-9]{3})*,[0-9]{2})`)

// BobsElektroSource checks bobselektro.de product pages. Not anti-bot protected;
// fetched directly.
type BobsElektroSource struct {
	client *http.Client
	fs     *FlareSolverr // optional
	urls   []string
}

// NewBobsElektro builds a bobselektro.de source for the given product URLs.
func NewBobsElektro(client *http.Client, fs *FlareSolverr, urls []string) *BobsElektroSource {
	return &BobsElektroSource{client: client, fs: fs, urls: urls}
}

func (s *BobsElektroSource) Name() string { return "bobselektro" }

func (s *BobsElektroSource) Check(ctx context.Context) ([]model.Availability, error) {
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
	if len(out) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

func (s *BobsElektroSource) checkOne(ctx context.Context, url string) (*model.Availability, error) {
	headers := map[string]string{"User-Agent": browserUserAgent, "Accept": "text/html,application/xhtml+xml", "Accept-Language": "de-DE,de;q=0.9"}
	body, err := getBody(ctx, s.client, s.fs, url, headers)
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	html := strings.ToLower(string(body))

	// The first delivery badge is the main product's; red means not orderable.
	m := bobDeliveryRe.FindStringSubmatch(html)
	if m == nil || m[1] == "red" {
		return nil, nil
	}

	var price *float64
	if pm := bobPriceRe.FindStringSubmatch(html); pm != nil {
		price = parseAmount(pm[1])
	}

	return &model.Availability{
		Source:      s.Name(),
		StoreName:   "Bobs Elektro",
		ProductName: "Midea PortaSplit",
		Stock:       1,
		Price:       price,
		URL:         url,
		Location:    "Online",
		Channel:     model.ChannelOnline,
		Key:         s.Name() + ":" + url,
	}, nil
}
