package metrics

import "context"

// Collector defines the contract for future metrics collection integrations.
type Collector interface {
	Collect(ctx context.Context) error
}
