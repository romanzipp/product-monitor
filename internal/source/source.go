// Package source contains the implementations of individual data sources.
package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const userAgent = "portasplit-monitor/1.0"

// browserUserAgent is for sources that reject non-browser clients with 404/403.
const browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// getBody fetches url, routing through FlareSolverr when fs is non-nil.
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
