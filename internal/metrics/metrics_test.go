package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExposition(t *testing.T) {
	m := New()
	price := 799.0
	m.ObserveCheck("braucheklima", true, 2, 5, &price)
	m.ObserveCheck("amazon", true, 1, 1, nil) // no price
	m.ObserveCheck("euronics", false, 0, 0, nil)
	m.ObserveNotification("braucheklima")

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	want := []string{
		`portasplit_source_up{source="braucheklima"} 1`,
		`portasplit_source_up{source="euronics"} 0`,
		`portasplit_source_stock{source="braucheklima"} 5`,
		`portasplit_source_available{source="amazon"} 1`,
		`portasplit_source_min_price_euros{source="braucheklima"} 799`,
		`portasplit_source_checks_total{source="euronics",result="error"} 1`,
		`portasplit_source_notifications_total{source="braucheklima"} 1`,
		`# TYPE portasplit_source_up gauge`,
		`# TYPE portasplit_source_checks_total counter`,
	}
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("missing line: %s", w)
		}
	}

	// Amazon has no price, so no min_price sample should be emitted for it.
	if strings.Contains(body, `portasplit_source_min_price_euros{source="amazon"}`) {
		t.Errorf("amazon should have no min_price sample")
	}
}
