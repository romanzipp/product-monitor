package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"portasplit-monitor/internal/model"
)

func captureForm(t *testing.T, p *Pushover) url.Values {
	t.Helper()
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.Form
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	old := pushoverEndpoint
	pushoverEndpoint = srv.URL
	defer func() { pushoverEndpoint = old }()

	if err := p.Notify(context.Background(), model.Availability{StoreName: "MediaMarkt", Source: "mediamarkt"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	return got
}

func TestEmergencyParams(t *testing.T) {
	// retry below 30 and expire above 10800 must be clamped.
	p := NewPushover(http.DefaultClient, "tok", "usr", 2, "", 5, 99999)
	f := captureForm(t, p)
	if f.Get("priority") != "2" {
		t.Errorf("priority = %q, want 2", f.Get("priority"))
	}
	if f.Get("retry") != "30" {
		t.Errorf("retry = %q, want 30 (clamped)", f.Get("retry"))
	}
	if f.Get("expire") != "10800" {
		t.Errorf("expire = %q, want 10800 (clamped)", f.Get("expire"))
	}
}

func TestNonEmergencyOmitsRetryExpire(t *testing.T) {
	p := NewPushover(http.DefaultClient, "tok", "usr", 0, "", 60, 3600)
	f := captureForm(t, p)
	if f.Has("retry") || f.Has("expire") {
		t.Errorf("non-emergency should not set retry/expire, got retry=%q expire=%q", f.Get("retry"), f.Get("expire"))
	}
}
