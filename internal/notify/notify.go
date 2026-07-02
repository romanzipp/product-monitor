// Package notify sends availability notifications via Pushover.
package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"product-monitor/internal/model"
)

// pushoverEndpoint is a var so tests can point it at a stub server.
var pushoverEndpoint = "https://api.pushover.net/1/messages.json"

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
	retry    int // emergency: seconds between repeats (>= 30)
	expire   int // emergency: seconds until repeats stop (<= 10800)
}

// NewPushover constructs a Pushover notifier. retry/expire (in seconds) apply
// only to emergency priority (2): the alert repeats every retry up to expire
// until acknowledged.
func NewPushover(client *http.Client, token, user string, priority int, device string, retry, expire int) *Pushover {
	return &Pushover{
		client:   client,
		token:    token,
		user:     user,
		priority: priority,
		device:   device,
		retry:    retry,
		expire:   expire,
	}
}

// Notify sends a single availability as a Pushover message.
func (p *Pushover) Notify(ctx context.Context, a model.Availability) error {
	form := url.Values{}
	form.Set("token", p.token)
	form.Set("user", p.user)
	form.Set("title", truncate(fmt.Sprintf("PortaSplit verfügbar: %s 🔥", a.StoreName), 250))
	form.Set("message", formatMessage(a))
	form.Set("priority", strconv.Itoa(p.priority))
	if p.priority >= 2 {
		// Emergency priority requires retry/expire; clamp to Pushover's limits.
		retry := p.retry
		if retry < 30 {
			retry = 30
		}
		expire := p.expire
		if expire > 10800 {
			expire = 10800
		}
		if expire < retry {
			expire = retry
		}
		form.Set("retry", strconv.Itoa(retry))
		form.Set("expire", strconv.Itoa(expire))
	}
	if p.device != "" {
		form.Set("device", p.device)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pushoverEndpoint, strings.NewReader(form.Encode()))
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
	if a.PreOrder {
		fmt.Fprintf(&b, "⏳ Vorbestellung / lange Lieferzeit\n")
	}
	if a.Price != nil {
		fmt.Fprintf(&b, "Price: %.2f €\n", *a.Price)
	} else {
		fmt.Fprintf(&b, "Price: n/a\n")
	}
	fmt.Fprintf(&b, "Stock: %d\n", a.Stock)
	// For in-store pickup, name the branch (the title only has the store name).
	if a.Channel == model.ChannelInStore && a.Location != "" {
		fmt.Fprintf(&b, "Filiale: %s\n", a.Location)
	}
	if a.URL != "" {
		fmt.Fprintf(&b, "%s via %s", a.URL, a.Source)
	} else {
		fmt.Fprintf(&b, "via %s", a.Source)
	}
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
