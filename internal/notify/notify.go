// Package notify sends availability notifications via Pushover.
package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"portasplit-monitor/internal/model"
)

const pushoverEndpoint = "https://api.pushover.net/1/messages.json"

// Notifier is implemented by any notification backend.
type Notifier interface {
	Notify(ctx context.Context, a model.Availability) error
}

// Pushover sends notifications through the Pushover API.
type Pushover struct {
	client   *http.Client
	token    string
	user     string
	priority int
	device   string
}

// NewPushover constructs a Pushover notifier.
func NewPushover(client *http.Client, token, user string, priority int, device string) *Pushover {
	return &Pushover{
		client:   client,
		token:    token,
		user:     user,
		priority: priority,
		device:   device,
	}
}

// Notify sends a single availability as a Pushover message.
func (p *Pushover) Notify(ctx context.Context, a model.Availability) error {
	form := url.Values{}
	form.Set("token", p.token)
	form.Set("user", p.user)
	form.Set("title", truncate("PortaSplit available: "+a.StoreName, 250))
	form.Set("message", formatMessage(a))
	form.Set("priority", strconv.Itoa(p.priority))
	if p.device != "" {
		form.Set("device", p.device)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushoverEndpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("pushover request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pushover returned status %d", resp.StatusCode)
	}
	return nil
}

func formatMessage(a model.Availability) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", a.ProductName)
	if a.Location != "" {
		fmt.Fprintf(&b, "%s — %s\n", a.StoreName, a.Location)
	} else {
		fmt.Fprintf(&b, "%s\n", a.StoreName)
	}
	fmt.Fprintf(&b, "Stock: %d\n", a.Stock)
	if a.Price != nil {
		fmt.Fprintf(&b, "Price: %.2f €\n", *a.Price)
	}
	if a.URL != "" {
		fmt.Fprintf(&b, "%s\n", a.URL)
	}
	fmt.Fprintf(&b, "via %s", a.Source)
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
