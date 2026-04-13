package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/shidaxi/go-webhook/internal/handler"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func setupBenchStore(b *testing.B, targetURL string, ruleCount int) *engine.RuleStore {
	b.Helper()
	rules := make([]config.Rule, ruleCount)
	for i := range rules {
		rules[i] = config.Rule{
			Name:  fmt.Sprintf("bench-rule-%d", i),
			Match: `len(payload.alerts) > 0`,
			Target: config.RuleTarget{
				URL:     fmt.Sprintf(`"%s"`, targetURL),
				Method:  "POST",
				Timeout: 5 * time.Second,
			},
			Body: `{"msg_type": "text", "content": payload.alerts[0].labels.alertname}`,
		}
	}
	store := engine.NewRuleStore()
	store.SetRules(engine.CompileRules(rules))
	return store
}

var benchPayload = map[string]any{
	"version": "4",
	"status":  "firing",
	"alerts": []any{
		map[string]any{
			"status": "firing",
			"labels": map[string]any{
				"alertname": "HighCPU",
				"severity":  "warning",
				"instance":  "web-01",
			},
			"annotations": map[string]any{
				"summary": "CPU usage > 80%",
			},
		},
	},
}

// BenchmarkWebhookHandler_SingleRule benchmarks with 1 matching rule.
func BenchmarkWebhookHandler_SingleRule(b *testing.B) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	store := setupBenchStore(b, target.URL, 1)
	r := gin.New()
	h := handler.NewWebhookHandler(store)
	r.POST("/webhook", h.Handle)

	body, _ := json.Marshal(benchPayload)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				b.Fatalf("unexpected status: %d", w.Code)
			}
		}
	})
}

// BenchmarkWebhookHandler_10Rules benchmarks with 10 matching rules dispatched concurrently.
func BenchmarkWebhookHandler_10Rules(b *testing.B) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	store := setupBenchStore(b, target.URL, 10)
	r := gin.New()
	h := handler.NewWebhookHandler(store)
	r.POST("/webhook", h.Handle)

	body, _ := json.Marshal(benchPayload)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				b.Fatalf("unexpected status: %d", w.Code)
			}
		}
	})
}

// BenchmarkWebhookHandler_NoMatch benchmarks the fast path where no rules match.
func BenchmarkWebhookHandler_NoMatch(b *testing.B) {
	store := engine.NewRuleStore()
	rules := []config.Rule{
		{
			Name:  "no-match",
			Match: `false`,
			Target: config.RuleTarget{
				URL:     `"http://localhost:1"`,
				Method:  "POST",
				Timeout: 5 * time.Second,
			},
			Body: `{"ok": true}`,
		},
	}
	store.SetRules(engine.CompileRules(rules))

	r := gin.New()
	h := handler.NewWebhookHandler(store)
	r.POST("/webhook", h.Handle)

	body, _ := json.Marshal(benchPayload)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
		}
	})
}

