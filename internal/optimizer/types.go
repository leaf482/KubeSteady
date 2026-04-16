package optimizer

import (
	"sort"

	"kubesteady/internal/metrics"
)

type Recommendation struct {
	Pod    string
	Action string // "scale_up" | "scale_down" | "no_op"
	Reason string
}

type Recommender struct{}

func (r Recommender) Recommend(usages []metrics.SmoothedCPUUsage) []Recommendation {
	recommendations := make([]Recommendation, 0, len(usages))

	for _, usage := range usages {
		action := "no_op"
		reason := "within threshold"

		if usage.CPU > 0.75 {
			action = "scale_up"
			reason = "cpu high"
		} else if usage.CPU < 0.25 {
			action = "scale_down"
			reason = "cpu low"
		}

		recommendations = append(recommendations, Recommendation{
			Pod:    usage.Pod,
			Action: action,
			Reason: reason,
		})
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Pod < recommendations[j].Pod
	})

	return recommendations
}
