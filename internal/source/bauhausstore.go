package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"product-monitor/internal/model"
)

const bauhausStoreProduct = "31934233"
const bauhausStorePDP = "https://www.bauhaus.info/klimaanlagen/midea-klimasplitgeraet-portasplit-12000-btu/p/31934233"
const bauhausSessionTTL = 15 * time.Minute

// BauhausStoreSource checks in-store (pickup) availability for one Bauhaus store
// via the /api/purchasability endpoint. The endpoint sits behind Cloudflare and
// only answers XHR requests, so a FlareSolverr session (cf_clearance cookie +
// user agent) is harvested and the API is then called directly. Replay requires
// the app and FlareSolverr to share an egress IP (cf_clearance is IP-bound).
type BauhausStoreSource struct {
	client    *http.Client
	fs        *FlareSolverr
	productID string
	storeID   string
	storeName string

	mu        sync.Mutex
	cookie    string
	userAgent string
	sessionAt time.Time
}

// NewBauhausStore builds a source for one Bauhaus store. It requires FlareSolverr.
func NewBauhausStore(client *http.Client, fs *FlareSolverr, storeID, storeName string) *BauhausStoreSource {
	return &BauhausStoreSource{
		client:    client,
		fs:        fs,
		productID: bauhausStoreProduct,
		storeID:   storeID,
		storeName: storeName,
	}
}

func (s *BauhausStoreSource) Name() string { return "bauhaus-store" }

func (s *BauhausStoreSource) Check(ctx context.Context) ([]model.Availability, error) {
	pr, err := s.query(ctx, false)
	if err != nil {
		// A stale/invalid session yields a 403 HTML page; refresh and retry once.
		pr, err = s.query(ctx, true)
		if err != nil {
			return nil, err
		}
	}

	return s.build(pr), nil
}

// build maps a purchasability response to in-store availability, keeping only the
// STORE result when it is purchasable.
func (s *BauhausStoreSource) build(pr *bauhausPurchasability) []model.Availability {
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
			URL:         bauhausStorePDP,
			Location:    s.storeName,
			Channel:     model.ChannelInStore,
			Targeted:    true,
			Key:         "bauhaus-store:" + s.storeID + ":" + s.productID,
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

// query calls the purchasability API with the (cached) FlareSolverr session.
func (s *BauhausStoreSource) query(ctx context.Context, refresh bool) (*bauhausPurchasability, error) {
	cookie, ua, err := s.session(ctx, refresh)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://www.bauhaus.info/api/purchasability?productId=%s&quantity=1&storeId=%s", s.productID, s.storeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9")
	req.Header.Set("Referer", bauhausStorePDP)
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
	cookie, ua, err := s.fs.Session(ctx, bauhausStorePDP)
	if err != nil {
		return "", "", err
	}
	s.cookie, s.userAgent, s.sessionAt = cookie, ua, time.Now()
	return cookie, ua, nil
}
