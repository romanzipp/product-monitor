package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"product-monitor/internal/model"
)

// expertWebcodeRe pulls the numeric web-code (article key) from an expert.de
// product URL, e.g. ".../klimagerate/32750011559-portasplit-….html" -> "32750011559".
var expertWebcodeRe = regexp.MustCompile(`(\d{9,})-`)

// ExpertSource checks expert.de online availability. The product page carries no
// schema.org data; stock and price come from expert's price API keyed by web-code
// and a store id. onlineButtonAction "ORDER" (or a positive onlineStock) means
// buyable online; "ORDER_IN_ADVANCE" / a PREORDER availability is a pre-order.
type ExpertSource struct {
	client  *http.Client
	baseURL string
	urls    []string
	storeID string
}

// NewExpert builds an expert.de source for the given product URLs. storeID is any
// valid expert store id (the online stock it returns is store-independent).
func NewExpert(client *http.Client, urls []string, storeID string) *ExpertSource {
	return &ExpertSource{
		client:  client,
		baseURL: "https://production.brntgs.expert.de/api/pricepds",
		urls:    urls,
		storeID: storeID,
	}
}

func (s *ExpertSource) Name() string { return "expert" }

func (s *ExpertSource) Check(ctx context.Context) ([]model.Availability, error) {
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

func (s *ExpertSource) checkOne(ctx context.Context, url string) (*model.Availability, error) {
	m := expertWebcodeRe.FindStringSubmatch(url)
	if m == nil {
		return nil, fmt.Errorf("no web-code in url %q", url)
	}
	webcode := m[1]

	endpoint := fmt.Sprintf("%s?webcode=%s&storeId=%s", s.baseURL, webcode, s.storeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Referer", "https://www.expert.de/")
	req.Header.Set("x-bt-reff", "checkout")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var res expertResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	p := res.Price

	orderable := p.OnlineStock > 0 || p.OnlineButtonAction == "ORDER" || p.OnlineButtonAction == "ORDER_IN_ADVANCE"
	if !orderable {
		return nil, nil
	}

	stock := p.OnlineStock
	if stock < 1 {
		stock = 1
	}

	return &model.Availability{
		Source:      s.Name(),
		StoreName:   "expert",
		ProductName: "Midea PortaSplit",
		Stock:       stock,
		Price:       p.BruttoPrice,
		URL:         url,
		Location:    "Online",
		Channel:     model.ChannelOnline,
		PreOrder:    p.OnlineStock == 0 || p.OnlineAvailability != "AVAILABLE",
		Key:         s.Name() + ":" + webcode,
	}, nil
}

type expertResponse struct {
	Price struct {
		OnlineStock        int      `json:"onlineStock"`
		OnlineAvailability string   `json:"onlineAvailability"`
		OnlineButtonAction string   `json:"onlineButtonAction"`
		BruttoPrice        *float64 `json:"bruttoPrice"`
	} `json:"price"`
}
