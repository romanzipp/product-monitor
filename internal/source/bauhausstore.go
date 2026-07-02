package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"product-monitor/internal/model"
)

// bauhausStoreReferer is a same-origin Bauhaus product page, used as the click-through
// URL in notifications (the monitored products come from config as productIDs).
const bauhausStoreReferer = "https://www.bauhaus.info/klimaanlagen/midea-klimasplitgeraet-portasplit-12000-btu/p/31934233"

// BauhausStore is one physical Bauhaus store to check (numeric id + display name).
type BauhausStore struct {
	ID   string
	Name string
}

// BauhausStoreSource checks in-store (pickup) availability for Bauhaus products and
// stores via the /api/purchasability endpoint (one call per product×store). The
// endpoint sits behind Cloudflare, so every call is routed through FlareSolverr,
// whose real browser carries the required TLS fingerprint and cf_clearance.
type BauhausStoreSource struct {
	client     *http.Client // unused; kept for a uniform constructor signature
	fs         *FlareSolverr
	productIDs []string
	stores     []BauhausStore
}

// NewBauhausStore builds a source for the given Bauhaus products and stores.
// Needs FlareSolverr.
func NewBauhausStore(client *http.Client, fs *FlareSolverr, productIDs []string, stores []BauhausStore) *BauhausStoreSource {
	return &BauhausStoreSource{
		client:     client,
		fs:         fs,
		productIDs: productIDs,
		stores:     stores,
	}
}

func (s *BauhausStoreSource) Name() string { return "bauhaus-store" }

func (s *BauhausStoreSource) Check(ctx context.Context) ([]model.Availability, error) {
	out := make([]model.Availability, 0, len(s.stores))
	var errs []error
	for _, productID := range s.productIDs {
		for _, store := range s.stores {
			pr, err := s.query(ctx, productID, store.ID)
			if err != nil {
				errs = append(errs, fmt.Errorf("store %s: %w", store.ID, err))
				continue
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

// query calls the purchasability API for one product+store through FlareSolverr.
func (s *BauhausStoreSource) query(ctx context.Context, productID, storeID string) (*bauhausPurchasability, error) {
	url := fmt.Sprintf("https://www.bauhaus.info/api/purchasability?productId=%s&quantity=1&storeId=%s", productID, storeID)
	body, err := s.fs.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var pr bauhausPurchasability
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &pr, nil
}
