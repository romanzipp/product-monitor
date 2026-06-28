package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

// FlareSolverr is a thin client for a FlareSolverr proxy
// (https://github.com/FlareSolverr/FlareSolverr). FlareSolverr drives a real
// browser to solve Cloudflare challenges, which lets sources behind Cloudflare
// (e.g. braucheklima.de, which 403s datacenter IPs) be fetched from a cluster.
type FlareSolverr struct {
	client     *http.Client
	endpoint   string        // base URL, e.g. http://flaresolverr:8191
	maxTimeout time.Duration // how long FlareSolverr may spend solving a challenge
}

// NewFlareSolverr builds a client for the FlareSolverr instance at endpoint
// (base URL without the /v1 path). It uses its own HTTP client whose timeout is
// always longer than maxTimeout, so the solver's internal deadline fires first.
func NewFlareSolverr(endpoint string, maxTimeout time.Duration) *FlareSolverr {
	return &FlareSolverr{
		client:     &http.Client{Timeout: maxTimeout + 15*time.Second},
		endpoint:   strings.TrimRight(endpoint, "/"),
		maxTimeout: maxTimeout,
	}
}

// Get fetches target through FlareSolverr and returns the response body. For a
// JSON endpoint the browser renders the payload inside a <pre> element, so the
// JSON is extracted from the returned HTML (see extractBody).
func (f *FlareSolverr) Get(ctx context.Context, target string) ([]byte, error) {
	payload, err := json.Marshal(fsRequest{
		Cmd:        "request.get",
		URL:        target,
		MaxTimeout: f.maxTimeout.Milliseconds(),
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint+"/v1", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flaresolverr request: %w", err)
	}
	defer resp.Body.Close()

	// FlareSolverr reports solve failures as HTTP 500 with the real reason in the
	// JSON body's "message" field, so decode the body regardless of status code.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("flaresolverr read: %w", err)
	}

	var fr fsResponse
	if err := json.Unmarshal(body, &fr); err != nil {
		return nil, fmt.Errorf("flaresolverr decode (HTTP %d): %w", resp.StatusCode, err)
	}
	if fr.Status != "ok" {
		msg := fr.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("flaresolverr: %s", msg)
	}
	if fr.Solution.Status != http.StatusOK {
		return nil, fmt.Errorf("flaresolverr upstream status %d", fr.Solution.Status)
	}

	return extractBody(fr.Solution.Response), nil
}

// extractBody pulls the actual payload out of FlareSolverr's rendered HTML. A
// raw JSON body is returned as-is; a body rendered by the browser's JSON viewer
// is wrapped in <pre>...</pre> with HTML-escaped content, which is unwrapped and
// unescaped here.
func extractBody(s string) []byte {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return []byte(trimmed)
	}

	open := strings.Index(s, "<pre")
	if open >= 0 {
		if gt := strings.IndexByte(s[open:], '>'); gt >= 0 {
			rest := s[open+gt+1:]
			if close := strings.Index(rest, "</pre>"); close >= 0 {
				return []byte(html.UnescapeString(strings.TrimSpace(rest[:close])))
			}
		}
	}
	return []byte(trimmed)
}

// --- FlareSolverr API types ---

type fsRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int64  `json:"maxTimeout"` // milliseconds
}

type fsResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		URL      string `json:"url"`
		Status   int    `json:"status"`
		Response string `json:"response"`
	} `json:"solution"`
}
