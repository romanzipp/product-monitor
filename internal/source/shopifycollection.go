package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"product-monitor/internal/model"
)

// ShopifyCollectionSource monitors an entire Shopify collection and alerts when
// ANY product in it has an available variant. It reads the collection's
// products.json (whose variants carry a reliable "available" flag, unlike the
// single-product .json), so no per-product config is needed. Each in-stock product
// is a separate result (StoreName = product title) with a link to that product.
type ShopifyCollectionSource struct {
	client    *http.Client
	fs        *FlareSolverr // optional
	urls      []string      // collection page URLs
	storeName string
}

// NewShopifyCollection builds a source for the given Shopify collection URLs.
func NewShopifyCollection(client *http.Client, fs *FlareSolverr, urls []string, storeName string) *ShopifyCollectionSource {
	return &ShopifyCollectionSource{client: client, fs: fs, urls: urls, storeName: storeName}
}

func (s *ShopifyCollectionSource) Name() string { return "shopify-collection" }

func (s *ShopifyCollectionSource) Check(ctx context.Context) ([]model.Availability, error) {
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

func (s *ShopifyCollectionSource) checkOne(ctx context.Context, collectionURL string) ([]model.Availability, error) {
	u, err := url.Parse(collectionURL)
	if err != nil {
		return nil, fmt.Errorf("parse url %q: %w", collectionURL, err)
	}
	base := u.Scheme + "://" + u.Host
	collection := strings.TrimRight(collectionURL, "/")
	headers := map[string]string{"User-Agent": browserUserAgent, "Accept": "application/json"}

	out := make([]model.Availability, 0)
	// products.json paginates at 250; loop until a short/empty page (cap for safety).
	for page := 1; page <= 20; page++ {
		endpoint := fmt.Sprintf("%s/products.json?limit=250&page=%d", collection, page)
		body, err := getBody(ctx, s.client, s.fs, endpoint, headers)
		if errors.Is(err, errNotFound) {
			break
		}
		if err != nil {
			return nil, err
		}
		var resp shopifyProducts
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("decode: %w", err)
		}
		if len(resp.Products) == 0 {
			break
		}
		for _, p := range resp.Products {
			price, available := minAvailablePrice(p.Variants)
			if !available {
				continue
			}
			out = append(out, model.Availability{
				Source:      s.Name(),
				StoreName:   p.Title,
				ProductName: p.Title,
				Stock:       1,
				Price:       price,
				URL:         base + "/products/" + p.Handle,
				Location:    s.storeName,
				Channel:     model.ChannelOnline,
				Key:         s.Name() + ":" + strconv.FormatInt(p.ID, 10),
			})
		}
		if len(resp.Products) < 250 {
			break
		}
	}
	return out, nil
}

type shopifyVariant struct {
	Available bool   `json:"available"`
	Price     string `json:"price"`
}

type shopifyProducts struct {
	Products []struct {
		ID       int64            `json:"id"`
		Title    string           `json:"title"`
		Handle   string           `json:"handle"`
		Variants []shopifyVariant `json:"variants"`
	} `json:"products"`
}

// minAvailablePrice returns the lowest price among available variants, and whether
// any variant is available.
func minAvailablePrice(variants []shopifyVariant) (*float64, bool) {
	var min *float64
	available := false
	for _, v := range variants {
		if !v.Available {
			continue
		}
		available = true
		if p := parseAmount(v.Price); p != nil && (min == nil || *p < *min) {
			min = p
		}
	}
	return min, available
}
