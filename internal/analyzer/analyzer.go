package analyzer

import "context"

// Analyzer defines the contract for future resource analysis logic.
type Analyzer interface {
	Analyze(ctx context.Context) error
}
