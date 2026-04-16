package observability

import (
	"time"

	"kubesteady/internal/metrics"
	"kubesteady/internal/optimizer"
)

type SystemSnapshot struct {
	Timestamp time.Time

	Pods int

	SmoothedCPU []metrics.SmoothedCPUUsage

	Recommendations []optimizer.Recommendation
	Validated       []optimizer.ValidatedRecommendation

	Rollbacks []optimizer.EvaluationResult
}

type SnapshotStore struct {
	current SystemSnapshot
}

func (s *SnapshotStore) Update(snapshot SystemSnapshot) {
	s.current = snapshot
}

func (s *SnapshotStore) Get() SystemSnapshot {
	return s.current
}
