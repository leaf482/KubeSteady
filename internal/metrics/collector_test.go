package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"kubesteady/internal/config"
)

func TestCollectKeepsValidSamplesWhenOneSeriesIsBad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"success",
			"data":{
				"resultType":"vector",
				"result":[
					{"metric":{"instance":"https://www.google.com"},"value":[1,"0.18"]},
					{"metric":{},"value":[1,"9"]},
					{"metric":{"instance":"https://open.spotify.com/"},"value":[1,"0.32"]}
				]
			}
		}`))
	}))
	defer server.Close()

	collector := NewPrometheusCollector(config.Config{
		PrometheusURL:   server.URL,
		PrometheusQuery: `avg_over_time(probe_duration_seconds[5m])`,
	})

	usages, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if collector.DataSource() != "prometheus" {
		t.Fatalf("source = %s, want prometheus", collector.DataSource())
	}
	if len(usages) != 2 {
		t.Fatalf("len = %d, want 2", len(usages))
	}
}

func TestCollectFallsBackWhenResultIsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer server.Close()

	collector := NewPrometheusCollector(config.Config{PrometheusURL: server.URL})
	usages, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if collector.DataSource() != "mock" || collector.FallbackReason() != "empty prometheus result" {
		t.Fatalf("source=%s reason=%s", collector.DataSource(), collector.FallbackReason())
	}
	if len(usages) != 3 {
		t.Fatalf("len = %d, want mock size 3", len(usages))
	}
}

func TestSmoothDropsDepartedPods(t *testing.T) {
	smoother := NewSmoother(0.5)
	first := smoother.Smooth([]WindowedSignal{{Pod: "a", AvgValue: 0.4}, {Pod: "b", AvgValue: 0.8}})
	if len(first) != 2 {
		t.Fatalf("first len = %d", len(first))
	}

	second := smoother.Smooth([]WindowedSignal{{Pod: "a", AvgValue: 0.2}})
	if len(second) != 1 || second[0].Pod != "a" {
		t.Fatalf("second = %+v", second)
	}
	if _, ok := smoother.state["b"]; ok {
		t.Fatal("departed pod b remained in EMA state")
	}
}
