package optimizer

import (
	"sort"
	"time"

	"kubesteady/internal/metrics"
)

type Recommendation struct {
	Pod        string
	Action     string // "scale_up" | "scale_down" | "no_op"
	Reason     string
	Confidence float64
}

type ValidatedRecommendation struct {
	Pod              string
	Action           string
	Reason           string
	Confidence       float64
	Valid            bool
	ValidationReason string
}

type Recommender struct{}
type Validator struct{}

type CooldownManager struct {
	window     time.Duration
	lastAction map[string]time.Time
}

func NewCooldownManager(window time.Duration) *CooldownManager {
	if window <= 0 {
		window = 2 * time.Minute
	}

	return &CooldownManager{
		window:     window,
		lastAction: make(map[string]time.Time),
	}
}

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

func (v Validator) Validate(recs []Recommendation) []ValidatedRecommendation {
	validated := make([]ValidatedRecommendation, 0, len(recs))

	for _, rec := range recs {
		result := ValidatedRecommendation{
			Pod:        rec.Pod,
			Action:     rec.Action,
			Reason:     rec.Reason,
			Confidence: rec.Confidence,
			Valid:      true,
		}

		if rec.Confidence < 0.5 {
			result.Valid = false
			result.Action = "no_op"
			result.ValidationReason = "low confidence"
		} else if rec.Action == "scale_down" && rec.Confidence < 0.7 {
			result.Valid = false
			result.Action = "no_op"
			result.ValidationReason = "unsafe scale down"
		}

		validated = append(validated, result)
	}

	return validated
}

func (c *CooldownManager) ApplyCooldown(recs []ValidatedRecommendation) []ValidatedRecommendation {
	now := time.Now()
	result := make([]ValidatedRecommendation, 0, len(recs))

	for _, rec := range recs {
		updated := rec

		if rec.Action == "no_op" {
			result = append(result, updated)
			continue
		}

		last, ok := c.lastAction[rec.Pod]
		if !ok {
			c.lastAction[rec.Pod] = now
			result = append(result, updated)
			continue
		}

		if now.Sub(last) < c.window {
			updated.Action = "no_op"
			updated.Valid = false
			updated.ValidationReason = "cooldown active"
			result = append(result, updated)
			continue
		}

		c.lastAction[rec.Pod] = now
		result = append(result, updated)
	}

	return result
}
