package source

import (
	"context"

	"product-monitor/internal/model"
)

// WithProduct wraps a source so every availability it returns carries the given
// product name and price cap. This lets one scraper serve differently-named
// products with different budgets and keeps both out of the source implementations
// (notifications show the name; the monitor applies the cap). Nothing keys dedup on
// ProductName, so overriding it here is safe. priceMax 0 = no limit.
func WithProduct(s model.Source, product string, priceMax int) model.Source {
	return &productNamed{Source: s, product: product, priceMax: priceMax}
}

type productNamed struct {
	model.Source
	product  string
	priceMax int
}

func (p *productNamed) Check(ctx context.Context) ([]model.Availability, error) {
	avail, err := p.Source.Check(ctx)
	for i := range avail {
		avail[i].ProductName = p.product
		avail[i].PriceMax = p.priceMax
	}
	return avail, err
}
