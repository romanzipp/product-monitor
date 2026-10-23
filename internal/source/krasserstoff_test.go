package source

import (
	"context"
	"testing"
)

func TestKrasserStoff(t *testing.T) {
	body := `{"data":{"event":{"id":"ev1","descriptiveName":"Show","available":true,"price":69.5,
		"resaleOffers":{"nodes":[
			{"id":"o1","price":69.5,"buyerFeeWithTaxes":3.5,"bought":false},
			{"id":"o2","price":69.5,"buyerFeeWithTaxes":3.5,"bought":true}
		]}}}}`
	srv := newPageServer(t, body)
	defer srv.Close()

	src := NewKrasserStoff(srv.Client(), "casperxo", []string{srv.URL + "/events/show?tab=resale"})
	got, err := src.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want regular + 1 open re:sale offer, got %d: %+v", len(got), got)
	}

	if got[0].Key != "krasserstoff:ev1" || got[0].URL != srv.URL+"/events/show" || *got[0].Price != 69.5 {
		t.Errorf("regular = %+v", got[0])
	}
	if got[1].Key != "krasserstoff:resale:o1" || got[1].URL != srv.URL+"/events/show?tab=resale" || *got[1].Price != 73 {
		t.Errorf("resale = %+v", got[1])
	}
}

func TestKrasserStoffSoldOut(t *testing.T) {
	srv := newPageServer(t, `{"data":{"event":{"id":"ev1","available":false,"price":69.5,"resaleOffers":{"nodes":[]}}}}`)
	defer srv.Close()

	got, err := NewKrasserStoff(srv.Client(), "casperxo", []string{srv.URL + "/events/show"}).Check(context.Background())
	if err != nil || len(got) != 0 {
		t.Fatalf("want empty, got %+v err=%v", got, err)
	}
}

func TestKrasserStoffUnknownEvent(t *testing.T) {
	srv := newPageServer(t, `{"data":{"event":null}}`)
	defer srv.Close()

	if _, err := NewKrasserStoff(srv.Client(), "casperxo", []string{srv.URL + "/events/nope"}).Check(context.Background()); err == nil {
		t.Fatal("want error for unknown event")
	}
}
