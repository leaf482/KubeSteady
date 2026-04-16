package observability

import (
	"sync"
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
	mu      sync.RWMutex
	current SystemSnapshot
}

func (s *SnapshotStore) Update(snapshot SystemSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = cloneSnapshot(snapshot)
}

func (s *SnapshotStore) Get() SystemSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSnapshot(s.current)
}

func cloneSnapshot(snapshot SystemSnapshot) SystemSnapshot {
	return SystemSnapshot{
		Timestamp:       snapshot.Timestamp,
		Pods:            snapshot.Pods,
		SmoothedCPU:     append([]metrics.SmoothedCPUUsage(nil), snapshot.SmoothedCPU...),
		Recommendations: append([]optimizer.Recommendation(nil), snapshot.Recommendations...),
		Validated:       append([]optimizer.ValidatedRecommendation(nil), snapshot.Validated...),
		Rollbacks:       append([]optimizer.EvaluationResult(nil), snapshot.Rollbacks...),
	}
}
