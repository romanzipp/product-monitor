package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"product-monitor/internal/model"
)

// ObiSource queries the OBI product availability API, which returns two arrays:
// `deliveryDataPerSeller` (online) and `pickupStores` (in-store). Entry shapes
// vary, so they are decoded into generic maps and read defensively.
type ObiSource struct {
	client     *http.Client
	baseURL    string // without the product id
	productID  string
	postalCode string
}

func NewObi(client *http.Client, productID, postalCode string) *ObiSource {
	return &ObiSource{
		client:     client,
		baseURL:    "https://www.obi.de/api/pdp/v1/availability",
		productID:  productID,
		postalCode: postalCode,
	}
}

func (s *ObiSource) Name() string { return "obi" }

func (s *ObiSource) Check(ctx context.Context) ([]model.Availability, error) {
	endpoint := fmt.Sprintf("%s/%s?postalCode=%s&quantity=1&lang=de-DE", s.baseURL, s.productID, s.postalCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var res obiResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	productName := "OBI #" + s.productID
	productURL := "https://www.obi.de"

	out := make([]model.Availability, 0)
	for _, st := range res.PickupStores {
		id := pickStr(st, "id", "storeId", "storeNo", "filialId")
		name := pickStr(st, "name", "storeName", "title", "city")
		qty := pickInt(st, "stock", "quantity", "availableStockQuantity", "availableQuantity")
		if qty < 1 {
			qty = 1 // listed implies available
		}
		storePLZ := firstNonEmpty(pickStr(st, "postalCode", "zipCode", "plz"), s.postalCode)
		out = append(out, model.Availability{
			Source:      s.Name(),
			StoreName:   firstNonEmpty(name, "OBI Filiale"),
			ProductName: productName,
			Stock:       qty,
			URL:         productURL,
			Location:    firstNonEmpty(pickStr(st, "city", "address", "street"), s.postalCode),
			Channel:     model.ChannelInStore,
			PLZ:         storePLZ,
			Key:         "obi:" + s.productID + ":pickup:" + id,
		})
	}
	for _, seller := range res.DeliveryDataPerSeller {
		id := pickStr(seller, "sellerId", "sellerName", "id")
		name := pickStr(seller, "sellerName", "seller", "name")
		qty := pickInt(seller, "stock", "quantity", "availableStockQuantity")
		if qty < 1 {
			qty = 1
		}
		out = append(out, model.Availability{
			Source:      s.Name(),
			StoreName:   firstNonEmpty(name, "OBI Online"),
			ProductName: productName,
			Stock:       qty,
			URL:         productURL,
			Location:    "Online · PLZ " + s.postalCode,
			Channel:     model.ChannelOnline,
			Key:         "obi:" + s.productID + ":delivery:" + id,
		})
	}
	return out, nil
}

type obiResponse struct {
	DeliveryDataPerSeller []map[string]any `json:"deliveryDataPerSeller"`
	PickupStores          []map[string]any `json:"pickupStores"`
}

// pickStr returns the first non-empty string value found under any of the keys.
func pickStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// pickInt returns the first integer value found under any of the keys.
func pickInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int:
				return n
			case string:
				if i, err := strconv.Atoi(n); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
