package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"portasplit-monitor/internal/model"
)

// BraucheKlimaSource reads the aggregated availability feed from braucheklima.de,
// a JSON array of stores each carrying an `articles` map keyed by product name.
// The feed sits behind Cloudflare and 403s datacenter IPs, so fs is used there.
type BraucheKlimaSource struct {
	client  *http.Client
	fs      *FlareSolverr // optional
	url     string
	product string // key in the `articles` map
}

// NewBraucheKlima constructs a source for the given product key. fs may be nil.
func NewBraucheKlima(client *http.Client, fs *FlareSolverr, url, product string) *BraucheKlimaSource {
	return &BraucheKlimaSource{client: client, fs: fs, url: url, product: product}
}

func (s *BraucheKlimaSource) Name() string { return "braucheklima" }

func (s *BraucheKlimaSource) Check(ctx context.Context) ([]model.Availability, error) {
	body, err := s.fetch(ctx)
	if err != nil {
		return nil, err
	}

	var stores []bkStore
	if err := json.Unmarshal(body, &stores); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	out := make([]model.Availability, 0)
	for _, st := range stores {
		art, ok := st.Articles[s.product]
		if !ok {
			continue
		}
		if len(art.Stocks) == 0 {
			continue
		}
		// Stocks are newest-first; index 0 is the current state.
		stock := art.Stocks[0].Stock
		if stock <= 0 {
			continue
		}

		var price *float64
		if len(art.Prices) > 0 {
			p := art.Prices[0].Price
			price = &p
		}

		// A postal code marks a physical store; online sellers have none.
		channel := model.ChannelOnline
		plz := ""
		if st.PLZ != nil && *st.PLZ != "" {
			channel = model.ChannelInStore
			plz = *st.PLZ
		}

		out = append(out, model.Availability{
			Source:      s.Name(),
			StoreName:   st.Name,
			ProductName: s.product,
			Stock:       stock,
			Price:       price,
			URL:         art.URL,
			Location:    bkLocation(st),
			Channel:     channel,
			PLZ:         plz,
			Key:         "braucheklima:" + strconv.Itoa(art.StoresArticlesID),
		})
	}
	return out, nil
}

func (s *BraucheKlimaSource) fetch(ctx context.Context) ([]byte, error) {
	headers := map[string]string{"Accept": "application/json", "User-Agent": userAgent}
	return getBody(ctx, s.client, s.fs, s.url, headers)
}

// bkLocation builds a location string, defaulting to "Online" without an address.
func bkLocation(s bkStore) string {
	var parts []string
	if s.Street != nil && *s.Street != "" {
		parts = append(parts, *s.Street)
	}
	city := strings.TrimSpace(coalesceStr(s.PLZ) + " " + coalesceStr(s.City))
	if city != "" {
		parts = append(parts, city)
	}
	if len(parts) == 0 {
		return "Online"
	}
	return strings.Join(parts, ", ")
}

func coalesceStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type bkStore struct {
	Name     string               `json:"name"`
	Lat      *float64             `json:"lat"`
	Lon      *float64             `json:"lon"`
	PLZ      *string              `json:"plz"`
	City     *string              `json:"city"`
	Street   *string              `json:"street"`
	Articles map[string]bkArticle `json:"articles"`
	Hash     string               `json:"hash"`
}

type bkArticle struct {
	StoresArticlesID int       `json:"storesArticlesId"`
	URL              string    `json:"url"`
	Stocks           []bkStock `json:"stocks"`
	// Prices may be JSON null; decoding into a nil slice handles that fine.
	Prices []bkPrice `json:"prices"`
}

type bkStock struct {
	Stock     int   `json:"stock"`
	Timestamp int64 `json:"timestamp"`
	LastSeen  int64 `json:"last_seen"`
}

type bkPrice struct {
	// Price is emitted as either a JSON integer or float, so float64 covers both.
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
	LastSeen  int64   `json:"last_seen"`
}
