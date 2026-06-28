package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"portasplit-monitor/internal/model"
)

// feedFixture mirrors the real braucheklima feed shape: an online seller with no
// address, a local physical store, and a far-away physical store.
const feedFixture = `[
  {"name":"Amazon","lat":null,"lon":null,"plz":null,"city":null,"street":null,
   "articles":{"Midea Portasplit":{"storesArticlesId":1,"url":"https://amazon.de/x",
     "stocks":[{"stock":3,"timestamp":2}],"prices":[{"price":699,"timestamp":2}]}}},
  {"name":"Globus Baumarkt Fulda","lat":50.5,"lon":9.7,"plz":"36100","city":"Fulda","street":"Justus-Liebig-Str. 9",
   "articles":{"Midea Portasplit":{"storesArticlesId":2,"url":"https://globus.de/x",
     "stocks":[{"stock":1,"timestamp":2}],"prices":[{"price":799,"timestamp":2}]}}},
  {"name":"Bauhaus Berlin-Pankow","lat":52.5,"lon":13.4,"plz":"13127","city":"Berlin","street":"Some Str. 1",
   "articles":{"Midea Portasplit":{"storesArticlesId":3,"url":"https://bauhaus.de/x",
     "stocks":[{"stock":2,"timestamp":2}],"prices":[]}}},
  {"name":"Toom Out","lat":null,"lon":null,"plz":"99999","city":"X","street":"Y",
   "articles":{"Midea Portasplit":{"storesArticlesId":4,"url":"https://toom.de/x",
     "stocks":[{"stock":0,"timestamp":2}],"prices":[]}}}
]`

func TestBraucheKlimaChannelClassification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(feedFixture))
	}))
	defer srv.Close()

	src := NewBraucheKlima(srv.Client(), nil, srv.URL, "Midea Portasplit")
	got, err := src.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	byStore := map[string]model.Availability{}
	for _, a := range got {
		byStore[a.StoreName] = a
	}

	// Out-of-stock store must be excluded.
	if len(got) != 3 {
		t.Fatalf("expected 3 in-stock results, got %d: %+v", len(got), got)
	}

	if a := byStore["Amazon"]; a.Channel != model.ChannelOnline || a.PLZ != "" {
		t.Errorf("Amazon: want online/empty PLZ, got %s/%q", a.Channel, a.PLZ)
	}
	if a := byStore["Globus Baumarkt Fulda"]; a.Channel != model.ChannelInStore || a.PLZ != "36100" {
		t.Errorf("Globus Fulda: want instore/36100, got %s/%q", a.Channel, a.PLZ)
	}
	if a := byStore["Bauhaus Berlin-Pankow"]; a.Channel != model.ChannelInStore || a.PLZ != "13127" {
		t.Errorf("Bauhaus Berlin: want instore/13127, got %s/%q", a.Channel, a.PLZ)
	}
}
