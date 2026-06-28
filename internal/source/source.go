// Package source contains the implementations of individual data sources.
// Each source implements model.Source. Adding a new source only requires
// implementing the interface and wiring an instance up in main.go.
package source

// userAgent identifies the monitor to upstream APIs.
const userAgent = "portasplit-monitor/1.0"

// browserUserAgent is used for sources (e.g. OBI) that reject non-browser
// clients with a 404/403. Identifying as a monitor is preferred where possible.
const browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
