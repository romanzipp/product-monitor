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

// grzInStockMaxDays is the delivery lead time (in Werktage) at or below which an
// item counts as immediately in stock; anything longer is treated as a pre-order.
const grzInStockMaxDays = 10

// grz-haustechnik.de (Shopware 5) emits schema.org/LimitedAvailability for every
// product regardless of real stock, so that marker is useless. The real signal is
// the delivery lead time "Lieferzeit <N> Werktage".
var grzLieferzeitRe = regexp.MustCompile(`lieferzeit\s+(\d+)\s+werktage`)

// grzIDRe pulls the numeric product id from a `…/<id>/<slug>` URL path. The id
// also appears on the page as `itemprop="productID"`, so it guards against soft-404s.
var grzIDRe = regexp.MustCompile(`/(\d+)/[^/]+/?$`)

// GrzSource checks grz-haustechnik.de product pages. Not anti-bot protected;
// fetched directly.
type GrzSource struct {
	client *http.Client
	fs     *FlareSolverr // optional
	urls   []string
}

// NewGrz builds a grz-haustechnik.de source for the given product URLs.
func NewGrz(client *http.Client, fs *FlareSolverr, urls []string) *GrzSource {
	return &GrzSource{client: client, fs: fs, urls: urls}
}

func (s *GrzSource) Name() string { return "grz" }

func (s *GrzSource) Check(ctx context.Context) ([]model.Availability, error) {
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

func (s *GrzSource) checkOne(ctx context.Context, url string) (*model.Availability, error) {
	headers := map[string]string{"User-Agent": browserUserAgent, "Accept": "text/html,application/xhtml+xml", "Accept-Language": "de-DE,de;q=0.9"}
	body, err := getBody(ctx, s.client, s.fs, url, headers)
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	html := strings.ToLower(string(body))

	// Soft-404 guard: the product id from the URL must be on the page.
	if m := grzIDRe.FindStringSubmatch(url); len(m) == 2 && !strings.Contains(html, m[1]) {
		return nil, nil
	}

	// Explicit sold-out states override the (constant) schema marker.
	for _, marker := range []string{"schema.org/outofstock", "schema.org/soldout", "nicht mehr verf", "ausverkauft"} {
		if strings.Contains(html, marker) {
			return nil, nil
		}
	}

	m := grzLieferzeitRe.FindStringSubmatch(html)
	if m == nil {
		// No delivery lead time means we can't confirm availability.
		return nil, nil
	}
	days, err := strconv.Atoi(m[1])
	if err != nil {
		return nil, nil
	}

	return &model.Availability{
		Source:      s.Name(),
		StoreName:   "GRZ Haustechnik",
		ProductName: "Midea PortaSplit",
		Stock:       1,
		Price:       parsePrice(html),
		URL:         url,
		Location:    "Online",
		Channel:     model.ChannelOnline,
		PreOrder:    days > grzInStockMaxDays,
		Key:         s.Name() + ":" + url,
	}, nil
}
