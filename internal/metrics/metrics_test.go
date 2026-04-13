package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestMetricsRegistered(t *testing.T) {
	// Verify all metrics are registered by collecting them
	ch := make(chan *prometheus.Metric, 10)

	// Counters and histograms should be describable
	descs := make(chan *prometheus.Desc, 10)

	RequestsTotal.Describe(descs)
	desc := <-descs
	assert.Contains(t, desc.String(), "webhook_requests_total")

	RequestDuration.Describe(descs)
	desc = <-descs
	assert.Contains(t, desc.String(), "webhook_request_duration_seconds")

	DispatchTotal.Describe(descs)
	desc = <-descs
	assert.Contains(t, desc.String(), "webhook_dispatch_total")

	DispatchDuration.Describe(descs)
	desc = <-descs
	assert.Contains(t, desc.String(), "webhook_dispatch_duration_seconds")

	RulesLoaded.Describe(descs)
	desc = <-descs
	assert.Contains(t, desc.String(), "webhook_rules_loaded")

	close(ch)
	close(descs)
}

func TestRequestsTotal_Increment(t *testing.T) {
	RequestsTotal.WithLabelValues("200", "/webhook").Inc()
	// No panic means it works with correct label count
}

func TestDispatchTotal_Increment(t *testing.T) {
	DispatchTotal.WithLabelValues("test-rule", "https://example.com", "200").Inc()
}

func TestRulesLoaded_Set(t *testing.T) {
	RulesLoaded.Set(5)
}
