package source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"product-monitor/internal/model"
)

// KrasserStoffSource monitors ticket events on Krasser Stoff shops (krasserstoff.com,
// shop.casperxo.com, ...). The HTML pages sit behind a Vercel bot checkpoint, but the
// shop's GraphQL API is reachable directly. It reports regular tickets when the event
// is available and one result per listed re:sale offer. resaleOffersAvailable is not
// used: it reads true while the page shows no offers.
type KrasserStoffSource struct {
	client *http.Client
	shop   string   // shop slug in the API path, e.g. "casperxo"
	urls   []string // event page URLs
}

// NewKrasserStoff builds a source for the given event URLs of one shop.
func NewKrasserStoff(client *http.Client, shop string, urls []string) *KrasserStoffSource {
	return &KrasserStoffSource{client: client, shop: shop, urls: urls}
}

func (s *KrasserStoffSource) Name() string { return "krasserstoff" }

func (s *KrasserStoffSource) Check(ctx context.Context) ([]model.Availability, error) {
	out := make([]model.Availability, 0)
	var errs []error
	for _, u := range s.urls {
		items, err := s.checkOne(ctx, u)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, items...)
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

const krasserStoffQuery = `query($id: ID!) { event(id: $id) { id descriptiveName available price resaleOffers(first: 50) { nodes { id price buyerFeeWithTaxes bought } } } }`

type krasserStoffResponse struct {
	Data struct {
		Event *struct {
			ID              string   `json:"id"`
			DescriptiveName string   `json:"descriptiveName"`
			Available       bool     `json:"available"`
			Price           *float64 `json:"price"`
			ResaleOffers    struct {
				Nodes []struct {
					ID                string  `json:"id"`
					Price             float64 `json:"price"`
					BuyerFeeWithTaxes float64 `json:"buyerFeeWithTaxes"`
					Bought            bool    `json:"bought"`
				} `json:"nodes"`
			} `json:"resaleOffers"`
		} `json:"event"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (s *KrasserStoffSource) checkOne(ctx context.Context, eventURL string) ([]model.Availability, error) {
	u, err := url.Parse(eventURL)
	if err != nil {
		return nil, fmt.Errorf("parse url %q: %w", eventURL, err)
	}

	base := u.Scheme + "://" + u.Host
	slug := strings.TrimPrefix(strings.TrimRight(u.Path, "/"), "/events/")
	pageURL := base + "/events/" + slug

	payload, err := json.Marshal(map[string]any{"query": krasserStoffQuery, "variables": map[string]string{"id": slug}})
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/v2/shops/%s/graphql?locale=de", base, s.shop)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: unexpected status %d", slug, resp.StatusCode)
	}

	var r krasserStoffResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if len(r.Errors) > 0 {
		return nil, fmt.Errorf("%s: graphql: %s", slug, r.Errors[0].Message)
	}

	ev := r.Data.Event
	if ev == nil {
		return nil, fmt.Errorf("%s: event not found", slug)
	}

	out := make([]model.Availability, 0)
	if ev.Available {
		out = append(out, model.Availability{
			Source:    s.Name(),
			StoreName: "Tickets " + ev.DescriptiveName,
			Stock:     1,
			Price:     ev.Price,
			URL:       pageURL,
			Channel:   model.ChannelOnline,
			Key:       s.Name() + ":" + ev.ID,
		})
	}

	for _, o := range ev.ResaleOffers.Nodes {
		if o.Bought {
			continue
		}

		// ponytail: assumes the buyer pays price + fee; unverified until a live offer is seen.
		price := o.Price + o.BuyerFeeWithTaxes
		out = append(out, model.Availability{
			Source:    s.Name(),
			StoreName: "re:sale " + ev.DescriptiveName,
			Stock:     1,
			Price:     &price,
			URL:       pageURL + "?tab=resale",
			Channel:   model.ChannelOnline,
			Key:       s.Name() + ":resale:" + o.ID,
		})
	}
	return out, nil
}
