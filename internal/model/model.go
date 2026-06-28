// Package model defines the core domain types shared across the application.
package model

import "context"

// Channel distinguishes how an item is available: shipped online vs. physically
// stocked in a local store for pickup.
type Channel string

const (
	ChannelOnline  Channel = "online"  // available for online order/delivery
	ChannelInStore Channel = "instore" // available in a physical store
)

// Availability describes a single in-stock observation for a product.
// It is produced by a Source and consumed by the monitor and notifier.
type Availability struct {
	Source      string   // logical name of the source, e.g. "braucheklima"
	StoreName   string   // human-readable store/location name
	ProductName string   // product name to display
	Stock       int      // number of units available (>0 means in stock)
	Price       *float64 // optional price in EUR
	URL         string   // direct link to the product, if known
	Location    string   // human-readable location (city/address/Online)
	Channel     Channel  // online vs in-store
	PLZ         string   // postal code for in-store items (empty for online)
	Key         string   // stable, unique dedup key (source + location + product)
}

// Source is the abstraction every data source implements. New sources can be
// added by implementing this interface and registering an instance in main.
type Source interface {
	// Name returns a short, unique, lowercase logical identifier.
	Name() string
	// Check queries the source and returns all currently available items.
	// An empty (non-nil) slice means "checked successfully, nothing in stock".
	Check(ctx context.Context) ([]Availability, error)
}
