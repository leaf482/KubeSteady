package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"kubesteady/internal/config"
)

type PodCPUUsage struct {
	Pod string
	CPU float64
}

type WindowedCPUUsage struct {
	Pod    string
	AvgCPU float64
}

type SmoothedCPUUsage struct {
	Pod string
	CPU float64
}

type Aggregator struct {
	window time.Duration
	data   map[string][]entry
}

type Smoother struct {
	alpha float64
	state map[string]float64
}

type entry struct {
	ts  time.Time
	cpu float64
}

func NewAggregator(window time.Duration) *Aggregator {
	if window <= 0 {
		window = 5 * time.Minute
	}

	return &Aggregator{
		window: window,
		data:   make(map[string][]entry),
	}
}

func NewSmoother(alpha float64) *Smoother {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.3
	}

	return &Smoother{
		alpha: alpha,
		state: make(map[string]float64),
	}
}

func (a *Aggregator) Aggregate(usages []PodCPUUsage) []WindowedCPUUsage {
	now := time.Now()
	cutoff := now.Add(-a.window)

	for pod, points := range a.data {
		filtered := points[:0]
		for _, p := range points {
			if !p.ts.Before(cutoff) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			delete(a.data, pod)
			continue
		}
		a.data[pod] = filtered
	}

	for _, usage := range usages {
		if usage.Pod == "" {
			continue
		}
		a.data[usage.Pod] = append(a.data[usage.Pod], entry{
			ts:  now,
			cpu: usage.CPU,
		})
	}

	pods := make([]string, 0, len(a.data))
	for pod := range a.data {
		pods = append(pods, pod)
	}
	sort.Strings(pods)

	out := make([]WindowedCPUUsage, 0, len(pods))
	for _, pod := range pods {
		points := a.data[pod]
		if len(points) == 0 {
			continue
		}
		var total float64
		for _, p := range points {
			total += p.cpu
		}
		out = append(out, WindowedCPUUsage{
			Pod:    pod,
			AvgCPU: total / float64(len(points)),
		})
	}

	return out
}

func (a *Aggregator) VarianceByPod() map[string]float64 {
	variances := make(map[string]float64, len(a.data))
	for pod, points := range a.data {
		variances[pod] = computeVariance(points)
	}
	return variances
}

func (s *Smoother) Smooth(usages []WindowedCPUUsage) []SmoothedCPUUsage {
	out := make([]SmoothedCPUUsage, 0, len(usages))
	for _, usage := range usages {
		if usage.Pod == "" {
			continue
		}

		current := usage.AvgCPU
		smoothed := current

		if prev, ok := s.state[usage.Pod]; ok {
			smoothed = (s.alpha * current) + ((1 - s.alpha) * prev)
		}

		s.state[usage.Pod] = smoothed
		out = append(out, SmoothedCPUUsage{
			Pod: usage.Pod,
			CPU: smoothed,
		})
	}

	return out
}

func computeVariance(entries []entry) float64 {
	if len(entries) == 0 {
		return 0
	}

	var sum float64
	for _, point := range entries {
		sum += point.cpu
	}
	mean := sum / float64(len(entries))

	var squaredDiffSum float64
	for _, point := range entries {
		diff := point.cpu - mean
		squaredDiffSum += diff * diff
	}

	return squaredDiffSum / float64(len(entries))
}

// Collector defines the contract for future metrics collection integrations.
type Collector interface {
	Collect(ctx context.Context) ([]PodCPUUsage, error)
}

type PrometheusCollector struct {
	baseURL        string
	query          string
	client         *http.Client
	lastDataSource string
}

var mockPodCPUUsage = []PodCPUUsage{
	{Pod: "pod-a", CPU: 0.62},
	{Pod: "pod-b", CPU: 0.81},
	{Pod: "pod-c", CPU: 0.21},
}

func NewPrometheusCollector(cfg config.Config) *PrometheusCollector {
	return &PrometheusCollector{
		baseURL: cfg.PrometheusURL,
		query:   cfg.PrometheusQuery,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		lastDataSource: "mock",
	}
}

func (c *PrometheusCollector) Collect(ctx context.Context) ([]PodCPUUsage, error) {
	if c.baseURL == "" {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}

	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}
	endpoint.Path = "/api/v1/query"

	params := endpoint.Query()
	params.Set("query", c.query)
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value []any `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}

	if payload.Status != "success" || payload.Data.ResultType != "vector" {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}
	if len(payload.Data.Result) == 0 {
		c.lastDataSource = "mock"
		return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
	}

	usages := make([]PodCPUUsage, 0, len(payload.Data.Result))
	for _, item := range payload.Data.Result {
		podName := podNameFromMetric(item.Metric)
		if podName == "" || len(item.Value) < 2 {
			c.lastDataSource = "mock"
			return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
		}

		valueRaw, ok := item.Value[1].(string)
		if !ok {
			c.lastDataSource = "mock"
			return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
		}

		cpu, err := strconv.ParseFloat(valueRaw, 64)
		if err != nil {
			c.lastDataSource = "mock"
			return append([]PodCPUUsage(nil), mockPodCPUUsage...), nil
		}

		usages = append(usages, PodCPUUsage{
			Pod: podName,
			CPU: cpu,
		})
	}

	c.lastDataSource = "prometheus"
	return usages, nil
}

func (c *PrometheusCollector) DataSource() string {
	return c.lastDataSource
}

func podNameFromMetric(metric map[string]string) string {
	if pod := metric["pod"]; pod != "" {
		return pod
	}
	if instance := metric["instance"]; instance != "" {
		return instance
	}
	if target := metric["target"]; target != "" {
		return target
	}
	return ""
}
