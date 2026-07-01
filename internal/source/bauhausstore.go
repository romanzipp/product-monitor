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

	"product-monitor/internal/model"
)

// bauhausStoreReferer is a same-origin Bauhaus page used to bootstrap the
// Cloudflare session (cf_clearance) and as the XHR Referer; it is not the
// monitored product (those come from config as productIDs).
const bauhausStoreReferer = "https://www.bauhaus.info/klimaanlagen/midea-klimasplitgeraet-portasplit-12000-btu/p/31934233"
const bauhausSessionTTL = 15 * time.Minute

// BauhausStoreSource checks in-store (pickup) availability for Bauhaus products
// and stores via the /api/purchasability endpoint (one call per product×store).
// The endpoint sits behind Cloudflare and only answers XHR requests, so a
// FlareSolverr session (cf_clearance cookie + user agent) is harvested and the
// API is then called directly. Replay requires the app and FlareSolverr to share
// an egress IP (cf_clearance is IP-bound).
type BauhausStoreSource struct {
	client     *http.Client
	fs         *FlareSolverr
	productIDs []string
	storeIDs   []string
	storeName  string

	mu        sync.Mutex
	cookie    string
	userAgent string
	sessionAt time.Time
}

// NewBauhausStore builds a source for the given Bauhaus products and stores.
// Needs FlareSolverr.
func NewBauhausStore(client *http.Client, fs *FlareSolverr, productIDs, storeIDs []string, storeName string) *BauhausStoreSource {
	return &BauhausStoreSource{
		client:     client,
		fs:         fs,
		productIDs: productIDs,
		storeIDs:   storeIDs,
		storeName:  storeName,
	}
}

func (s *BauhausStoreSource) Name() string { return "bauhaus-store" }

func (s *BauhausStoreSource) Check(ctx context.Context) ([]model.Availability, error) {
	out := make([]model.Availability, 0, len(s.storeIDs))
	var errs []error
	for _, productID := range s.productIDs {
		for _, storeID := range s.storeIDs {
			pr, err := s.query(ctx, productID, storeID, false)
			if err != nil {
				// A stale/invalid session yields a 403 HTML page; refresh and retry once.
				if pr, err = s.query(ctx, productID, storeID, true); err != nil {
					errs = append(errs, err)
					continue
				}
			}
			out = append(out, s.build(productID, storeID, pr)...)
		}
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// build maps a purchasability response to in-store availability, keeping only the
// STORE result when it is purchasable.
func (s *BauhausStoreSource) build(productID, storeID string, pr *bauhausPurchasability) []model.Availability {
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
			StoreName:   s.storeName,
			ProductName: "Midea PortaSplit",
			Stock:       stock,
			URL:         bauhausStoreReferer,
			Location:    s.storeName,
			Channel:     model.ChannelInStore,
			Targeted:    true,
			Key:         "bauhaus-store:" + storeID + ":" + productID,
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

// query calls the purchasability API for one product+store with the (cached) session.
func (s *BauhausStoreSource) query(ctx context.Context, productID, storeID string, refresh bool) (*bauhausPurchasability, error) {
	cookie, ua, err := s.session(ctx, refresh)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://www.bauhaus.info/api/purchasability?productId=%s&quantity=1&storeId=%s", productID, storeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9")
	req.Header.Set("Referer", bauhausStoreReferer)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Cookie", cookie)

	resp, err := s.client.Do(req)
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

// session returns a cached cookie header + user agent, harvesting a fresh one
// via FlareSolverr when forced or expired.
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
