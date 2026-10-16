package optimizer

import (
	"testing"
	"time"

	"kubesteady/internal/metrics"
)

func TestRecommendUsesLowConfidenceWithoutHistory(t *testing.T) {
	aggregator := metrics.NewAggregator(time.Minute)
	recs := Recommender{}.Recommend([]metrics.SmoothedCPUUsage{{
		Pod: "pod-a",
		CPU: 0.9,
	}}, aggregator)

	if len(recs) != 1 {
		t.Fatalf("len = %d", len(recs))
	}
	if recs[0].Confidence != 0.4 {
		t.Fatalf("confidence = %v, want 0.4", recs[0].Confidence)
	}
	if recs[0].Action != "scale_up" {
		t.Fatalf("action = %s", recs[0].Action)
	}
}

func TestRecommendLatencyThresholds(t *testing.T) {
	aggregator := metrics.NewAggregator(time.Minute)
	_ = aggregator.Aggregate([]metrics.PodCPUUsage{{Pod: "site", CPU: 0.6}})

	recs := Recommender{LatencyMode: true}.Recommend([]metrics.SmoothedCPUUsage{{
		Pod: "site",
		CPU: 0.6,
	}}, aggregator)

	if recs[0].Action != "scale_up" || recs[0].Reason != "high latency" {
		t.Fatalf("high latency rec = %+v", recs[0])
	}
	if recs[0].Confidence != 0.9 {
		t.Fatalf("confidence = %v, want 0.9 for a single stable sample", recs[0].Confidence)
	}
}
