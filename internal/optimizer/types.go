package optimizer

import (
	"sort"

	"kubesteady/internal/metrics"
)

type Recommendation struct {
	Pod        string
	Action     string // "scale_up" | "scale_down" | "no_op"
	Reason     string
	Confidence float64
}

type Recommender struct{}

func (r Recommender) Recommend(usages []metrics.SmoothedCPUUsage, aggregator *metrics.Aggregator) []Recommendation {
	recommendations := make([]Recommendation, 0, len(usages))
	variances := map[string]float64{}
	if aggregator != nil {
		variances = aggregator.VarianceByPod()
	}

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

		confidence := confidenceFromVariance(variances[usage.Pod])
		recommendations = append(recommendations, Recommendation{
			Pod:        usage.Pod,
			Action:     action,
			Reason:     reason,
			Confidence: confidence,
		})
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Pod < recommendations[j].Pod
	})

	return recommendations
}

func confidenceFromVariance(variance float64) float64 {
	if variance < 0.01 {
		return 0.9
	}
	if variance < 0.05 {
		return 0.7
	}
	return 0.4
}
