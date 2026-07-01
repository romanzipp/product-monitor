package source

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"regexp"
	"strings"

	"product-monitor/internal/model"
)

// weinmannAvailRe extracts the first `availabilityJson` blob from a page. The
// value is HTML-entity-encoded (&#34; for ") with the inner JSON quotes escaped as
// \"; both are decoded before parsing. The first blob is the main product's (it
// renders before any cross-sell cards).
var weinmannAvailRe = regexp.MustCompile(`availabilityJson":"(\{.*?\})"`)

// weinmannSKURe pulls the dashed article number (e.g. "90-134-79") from a URL.
var weinmannSKURe = regexp.MustCompile(`(\d+-\d+-\d+)\.html`)

// WeinmannSchanzSource checks weinmann-schanz.de, a B2B wholesaler (Magento PWA).
// It exposes no schema.org availability; stock lives in an `availabilityJson` blob
// with a numeric id. Only id 1 ("Sofort Lieferbar") is real stock; id 3 ("Derzeit
// nicht auf Lager - Lieferzeit ca. 5-7 Wochen") and id 7 ("Aktuell nicht lieferbar")
// are backorder/out-of-stock (their is_in_stock flag is unreliable). Prices are
// customer-group gated and not shown to anonymous visitors, so no price is
// reported. Not anti-bot protected.
type WeinmannSchanzSource struct {
	client *http.Client
	fs     *FlareSolverr // optional
	urls   []string
}

// NewWeinmannSchanz builds a weinmann-schanz.de source for the given product URLs.
func NewWeinmannSchanz(client *http.Client, fs *FlareSolverr, urls []string) *WeinmannSchanzSource {
	return &WeinmannSchanzSource{client: client, fs: fs, urls: urls}
}

func (s *WeinmannSchanzSource) Name() string { return "weinmannschanz" }

func (s *WeinmannSchanzSource) Check(ctx context.Context) ([]model.Availability, error) {
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

func (s *WeinmannSchanzSource) checkOne(ctx context.Context, url string) (*model.Availability, error) {
	headers := map[string]string{"User-Agent": browserUserAgent, "Accept": "text/html,application/xhtml+xml", "Accept-Language": "de-DE,de;q=0.9"}
	body, err := getBody(ctx, s.client, s.fs, url, headers)
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	decoded := html.UnescapeString(string(body))

	// Soft-404 guard: the article number (dashes stripped) must be on the page.
	if m := weinmannSKURe.FindStringSubmatch(url); len(m) == 2 {
		sku := strings.ReplaceAll(m[1], "-", "")
		if !strings.Contains(decoded, sku) {
			return nil, nil
		}
	}

	m := weinmannAvailRe.FindStringSubmatch(decoded)
	if m == nil {
		return nil, nil
	}
	var av struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.ReplaceAll(m[1], `\"`, `"`)), &av); err != nil {
		return nil, nil
	}
	// Only id 1 ("Sofort Lieferbar") is genuinely in stock. The is_in_stock flag is
	// unreliable: id 3 ("Derzeit nicht auf Lager - Lieferzeit ca. 5-7 Wochen") and
	// id 7 ("Aktuell nicht lieferbar") both mean not available, so they must not alert.
	if av.ID != 1 {
		return nil, nil
	}

	return &model.Availability{
		Source:      s.Name(),
		StoreName:   "Weinmann & Schanz",
		ProductName: "Midea PortaSplit",
		Stock:       1,
		URL:         url,
		Location:    "Online",
		Channel:     model.ChannelOnline,
		Key:         s.Name() + ":" + url,
	}, nil
}
