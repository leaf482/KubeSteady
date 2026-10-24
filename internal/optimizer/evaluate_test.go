package optimizer

import (
	"testing"

	"kubesteady/internal/metrics"
)

func TestEvaluateMarksSignalIncreaseAsDegraded(t *testing.T) {
	pre := []metrics.SmoothedSignal{{Pod: "site", Value: 0.2}}
	post := []metrics.SmoothedSignal{{Pod: "site", Value: 0.3}}

	results := Evaluator{}.Evaluate(pre, post)
	if len(results) != 1 {
		t.Fatalf("len = %d", len(results))
	}
	if !results[0].Degraded || results[0].Reason != "signal increased" {
		t.Fatalf("result = %+v", results[0])
	}
}
