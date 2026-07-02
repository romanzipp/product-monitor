package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"product-monitor/internal/model"
)

// bauhausStoreReferer is a same-origin Bauhaus product page: it is used to harvest
// the Cloudflare session (cf_clearance) via FlareSolverr, as the XHR Referer, and
// as the click-through URL in notifications (monitored products come from config).
const bauhausStoreReferer = "https://www.bauhaus.info/klimaanlagen/midea-klimasplitgeraet-portasplit-12000-btu/p/31934233"
const bauhausSessionTTL = 15 * time.Minute

// BauhausStore is one physical Bauhaus store to check (numeric id + display name).
type BauhausStore struct {
	ID   string
	Name string
}

// BauhausStoreSource checks in-store (pickup) availability for Bauhaus products and
// stores via the /api/purchasability endpoint (one call per product×store). The
// endpoint is JSON-over-XHR behind Cloudflare: FlareSolverr solves the challenge and
// yields a cf_clearance cookie, but the API also validates the TLS (JA3) fingerprint,
// so the call is replayed with a Chrome-impersonating tls-client rather than the Go
// stdlib client (which gets a 403). cf_clearance is IP-bound, so the app and
// FlareSolverr must share an egress IP.
type BauhausStoreSource struct {
	fs         *FlareSolverr
	productIDs []string
	stores     []BauhausStore
	tls        tls_client.HttpClient

	mu        sync.Mutex
	cookie    string
	userAgent string
	sessionAt time.Time
}

// NewBauhausStore builds a source for the given Bauhaus products and stores.
// Needs FlareSolverr.
func NewBauhausStore(client *http.Client, fs *FlareSolverr, productIDs []string, stores []BauhausStore) *BauhausStoreSource {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_133),
	}
	tlsClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		tlsClient = nil // query() surfaces this as an error per cycle
	}
	return &BauhausStoreSource{
		fs:         fs,
		productIDs: productIDs,
		stores:     stores,
		tls:        tlsClient,
	}
}

func (s *BauhausStoreSource) Name() string { return "bauhaus-store" }

func (s *BauhausStoreSource) Check(ctx context.Context) ([]model.Availability, error) {
	out := make([]model.Availability, 0, len(s.stores))
	var errs []error
	for _, productID := range s.productIDs {
		for _, store := range s.stores {
			pr, err := s.query(ctx, productID, store.ID, false)
			if err != nil {
				// A stale session yields a 403; refresh cf_clearance and retry once.
				if pr, err = s.query(ctx, productID, store.ID, true); err != nil {
					errs = append(errs, fmt.Errorf("store %s: %w", store.ID, err))
					continue
				}
			}
			out = append(out, s.build(productID, store, pr)...)
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// build maps a purchasability response to in-store availability, keeping only the
// STORE result when it is purchasable.
func (s *BauhausStoreSource) build(productID string, store BauhausStore, pr *bauhausPurchasability) []model.Availability {
	out := make([]model.Availability, 0, 1)
	for _, r := range pr.Results {
		if r.Kind != "STORE" || !r.Purchasable {
			continue
		}
		stock := r.Amount
		if stock < 1 {
			stock = 1
		}
		out = append(out, model.Availability{
			Source:      s.Name(),
			StoreName:   store.Name,
			ProductName: "Midea PortaSplit",
			Stock:       stock,
			URL:         bauhausStoreReferer,
			Location:    store.Name,
			Channel:     model.ChannelInStore,
			Targeted:    true,
			Key:         "bauhaus-store:" + store.ID + ":" + productID,
		})
	}
	return out
}

type bauhausPurchasability struct {
	Results []struct {
		Amount      int    `json:"amount"`
		Code        string `json:"code"`
		Kind        string `json:"kind"`
		Purchasable bool   `json:"purchasable"`
	} `json:"results"`
}

// query calls the purchasability API for one product+store, replaying the
// FlareSolverr-harvested Cloudflare session with a Chrome TLS fingerprint.
func (s *BauhausStoreSource) query(ctx context.Context, productID, storeID string, refresh bool) (*bauhausPurchasability, error) {
	if s.tls == nil {
		return nil, errors.New("tls-client unavailable")
	}
	cookie, ua, err := s.session(ctx, refresh)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://www.bauhaus.info/api/purchasability?productId=%s&quantity=1&storeId=%s", productID, storeID)
	req, err := fhttp.NewRequestWithContext(ctx, fhttp.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = fhttp.Header{
		"accept":           {"application/json, text/plain, */*"},
		"accept-language":  {"de-DE,de;q=0.9"},
		"user-agent":       {ua},
		"referer":          {bauhausStoreReferer},
		"x-requested-with": {"XMLHttpRequest"},
		"sec-fetch-dest":   {"empty"},
		"sec-fetch-mode":   {"cors"},
		"sec-fetch-site":   {"same-origin"},
		"cookie":           {cookie},
	}

	resp, err := s.tls.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("purchasability status %d", resp.StatusCode)
	}

	var pr bauhausPurchasability
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &pr, nil
}

// session returns a cached cookie header + user agent, harvesting a fresh one via
// FlareSolverr when forced or expired.
func (s *BauhausStoreSource) session(ctx context.Context, force bool) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !force && s.cookie != "" && time.Since(s.sessionAt) < bauhausSessionTTL {
		return s.cookie, s.userAgent, nil
	}
	cookie, ua, err := s.fs.Session(ctx, bauhausStoreReferer)
	if err != nil {
		return "", "", err
	}
	s.cookie, s.userAgent, s.sessionAt = cookie, ua, time.Now()
	return cookie, ua, nil
}
