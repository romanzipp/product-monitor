// Package model defines the core domain types shared across the application.
package model

import "context"

// Channel distinguishes online vs in-store availability.
type Channel string

const (
	ChannelOnline  Channel = "online"
	ChannelInStore Channel = "instore"
)

// Availability is a single in-stock observation for a product.
type Availability struct {
	Source      string
	StoreName   string
	ProductName string
	Stock       int
	Price       *float64
	URL         string
	Location    string
	Channel     Channel
	PLZ         string // empty for online
	Targeted    bool   // source already targets a specific store; skip local filter
	PreOrder    bool   // orderable but pre-order / long delivery, not immediately in stock
	Key         string // stable dedup key
}

// Source is the abstraction every data source implements.
type Source interface {
	Name() string
	// Check returns all currently available items. An empty (non-nil) slice
	// means checked successfully, nothing in stock.
	Check(ctx context.Context) ([]Availability, error)
}
