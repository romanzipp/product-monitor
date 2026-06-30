package source

import (
	"encoding/json"
	"testing"

	"portasplit-monitor/internal/model"
)

func TestBauhausStoreBuild(t *testing.T) {
	src := NewBauhausStore(nil, nil, "589", "Bauhaus Frankfurt")

	// Exact shape returned by the purchasability API (both out of stock).
	const oos = `{"results":[{"amount":0,"code":"OOOS","kind":"ONLINE","product":"31934233","purchasable":false},{"amount":0,"code":"SOOS","kind":"STORE","product":"31934233","purchasable":false}]}`
	const inStore = `{"results":[{"amount":0,"code":"OOOS","kind":"ONLINE","product":"31934233","purchasable":false},{"amount":3,"code":"IS","kind":"STORE","product":"31934233","purchasable":true}]}`

	parse := func(s string) *bauhausPurchasability {
		var pr bauhausPurchasability
		if err := json.Unmarshal([]byte(s), &pr); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return &pr
	}

	if got := src.build(parse(oos)); len(got) != 0 {
		t.Fatalf("out of stock: want 0, got %d: %+v", len(got), got)
	}

	got := src.build(parse(inStore))
	if len(got) != 1 {
		t.Fatalf("in store: want 1, got %d", len(got))
	}
	a := got[0]
	if a.Channel != model.ChannelInStore || !a.Targeted || a.Stock != 3 {
		t.Errorf("want instore/targeted/stock=3, got %s/%v/%d", a.Channel, a.Targeted, a.Stock)
	}
}
