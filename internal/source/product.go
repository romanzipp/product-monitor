package source

import (
	"context"

	"product-monitor/internal/model"
)

// WithProduct wraps a source so every availability it returns carries the given
// product name. This lets one scraper serve differently-named products and keeps
// the product name out of the source implementations (notifications show it).
// Nothing keys dedup on ProductName, so overriding it here is safe.
func WithProduct(s model.Source, product string) model.Source {
	return &productNamed{Source: s, product: product}
}

type productNamed struct {
	model.Source
	product string
}

func (p *productNamed) Check(ctx context.Context) ([]model.Availability, error) {
	avail, err := p.Source.Check(ctx)
	for i := range avail {
		avail[i].ProductName = p.product
	}
	return avail, err
}
