// Package source contains the implementations of individual data sources.
// Each source implements model.Source. Adding a new source only requires
// implementing the interface and wiring an instance up in main.go.
package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// userAgent identifies the monitor to upstream APIs.
const userAgent = "portasplit-monitor/1.0"

// browserUserAgent is used for sources (e.g. OBI) that reject non-browser
// clients with a 404/403. Identifying as a monitor is preferred where possible.
const browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// getBody fetches url and returns the response body. When fs is non-nil the
// request is routed through FlareSolverr (to bypass Cloudflare/anti-bot);
// otherwise a plain GET is made with the given headers.
func getBody(ctx context.Context, client *http.Client, fs *FlareSolverr, url string, headers map[string]string) ([]byte, error) {
	if fs != nil {
		return fs.Get(ctx, url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
