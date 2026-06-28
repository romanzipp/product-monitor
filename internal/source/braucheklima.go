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

// BraucheKlimaSource reads the aggregated availability feed published by
// braucheklima.de. The feed is a JSON array of stores, each carrying an
// `articles` map keyed by product name with time-series `stocks`/`prices`.
type BraucheKlimaSource struct {
	client  *http.Client
	url     string
	product string // exact key to look up in the `articles` map
}

// NewBraucheKlima constructs a source for the given product key.
func NewBraucheKlima(client *http.Client, url, product string) *BraucheKlimaSource {
	return &BraucheKlimaSource{client: client, url: url, product: product}
}

func (s *BraucheKlimaSource) Name() string { return "braucheklima" }

func (s *BraucheKlimaSource) Check(ctx context.Context) ([]model.Availability, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var stores []bkStore
	if err := json.NewDecoder(resp.Body).Decode(&stores); err != nil {
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
		// The stocks array is sorted newest-first; index 0 is the current state.
		stock := art.Stocks[0].Stock
		if stock <= 0 {
			continue
		}

		var price *float64
		if len(art.Prices) > 0 {
			p := art.Prices[0].Price
			price = &p
		}

		out = append(out, model.Availability{
			Source:      s.Name(),
			StoreName:   st.Name,
			ProductName: s.product,
			Stock:       stock,
			Price:       price,
			URL:         art.URL,
			Location:    bkLocation(st),
			Key:         "braucheklima:" + strconv.Itoa(art.StoresArticlesID),
		})
	}
	return out, nil
}

// bkLocation builds a human-readable location string, defaulting to "Online"
// for stores without a physical address (e.g. Amazon).
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

// --- response types ---

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
