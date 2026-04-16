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

type EvaluationResult struct {
	Pod            string
	ShouldRollback bool
	Reason         string
}

type Recommender struct {
	LatencyMode bool
}
type Validator struct{}
type Evaluator struct{}

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

		if r.LatencyMode {
			if usage.CPU > 0.5 {
				action = "scale_up"
				reason = "high latency"
			} else if usage.CPU < 0.2 {
				action = "no_op"
				reason = "latency healthy"
			} else {
				action = "no_op"
				reason = "latency stable"
			}
		} else {
			if usage.CPU > 0.75 {
				action = "scale_up"
				reason = "cpu high"
			} else if usage.CPU < 0.25 {
				action = "scale_down"
				reason = "cpu low"
			}
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

func (e Evaluator) Evaluate(pre []metrics.SmoothedCPUUsage, post []metrics.SmoothedCPUUsage) []EvaluationResult {
	preByPod := make(map[string]float64, len(pre))
	postByPod := make(map[string]float64, len(post))
	podSet := make(map[string]struct{}, len(pre)+len(post))

	for _, usage := range pre {
		if usage.Pod == "" {
			continue
		}
		preByPod[usage.Pod] = usage.CPU
		podSet[usage.Pod] = struct{}{}
	}

	for _, usage := range post {
		if usage.Pod == "" {
			continue
		}
		postByPod[usage.Pod] = usage.CPU
		podSet[usage.Pod] = struct{}{}
	}

	pods := make([]string, 0, len(podSet))
	for pod := range podSet {
		pods = append(pods, pod)
	}
	sort.Strings(pods)

	results := make([]EvaluationResult, 0, len(pods))
	for _, pod := range pods {
		preCPU, hasPre := preByPod[pod]
		postCPU, hasPost := postByPod[pod]

		if !hasPre || !hasPost {
			results = append(results, EvaluationResult{
				Pod:            pod,
				ShouldRollback: false,
				Reason:         "missing metrics",
			})
			continue
		}

		if postCPU > preCPU*1.2 {
			results = append(results, EvaluationResult{
				Pod:            pod,
				ShouldRollback: true,
				Reason:         "cpu increased",
			})
			continue
		}

		if postCPU < preCPU*0.8 {
			results = append(results, EvaluationResult{
				Pod:            pod,
				ShouldRollback: false,
				Reason:         "improved",
			})
			continue
		}

		results = append(results, EvaluationResult{
			Pod:            pod,
			ShouldRollback: false,
			Reason:         "no significant change",
		})
	}

	return results
}
