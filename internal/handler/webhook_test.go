package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupWebhookHandler(t *testing.T, rules []config.Rule, targetServer *httptest.Server) (*gin.Engine, *engine.RuleStore) {
	t.Helper()
	store := engine.NewRuleStore()

	// If a target server is provided, replace the URL expression to point to it
	if targetServer != nil {
		for i := range rules {
			rules[i].Target.URL = `"` + targetServer.URL + `"`
		}
	}

	compiled := engine.CompileRules(rules)
	store.SetRules(compiled)

	r := gin.New()
	h := NewWebhookHandler(store)
	r.POST("/webhook", h.Handle)
	return r, store
}

func TestWebhookHandler_MatchAndForward(t *testing.T) {
	var receivedBody map[string]any
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	rules := []config.Rule{
		{
			Name:  "test-forward",
			Match: `len(payload.alerts) > 0`,
			Target: config.RuleTarget{
				Method:  "POST",
				Timeout: 5000000000, // 5s in nanoseconds
			},
			Body: `{"msg_type": "text", "content": payload.alerts[0].labels.alertname}`,
		},
	}

	r, _ := setupWebhookHandler(t, rules, target)

	payload := map[string]any{
		"alerts": []any{
			map[string]any{
				"labels": map[string]any{
					"alertname": "HighCPU",
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text", receivedBody["msg_type"])
	assert.Equal(t, "HighCPU", receivedBody["content"])
}

func TestWebhookHandler_NoMatchingRules(t *testing.T) {
	rules := []config.Rule{
		{
			Name:  "never-match",
			Match: `false`,
			Target: config.RuleTarget{
				URL:    `"https://example.com"`,
				Method: "POST",
			},
			Body: `{"ok": true}`,
		},
	}

	r, _ := setupWebhookHandler(t, rules, nil)

	body := []byte(`{"alerts": []}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["matched"])
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	rules := []config.Rule{}
	r, _ := setupWebhookHandler(t, rules, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookHandler_MultipleRulesMatch(t *testing.T) {
	var callCount atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	rules := []config.Rule{
		{
			Name:  "rule-a",
			Match: `true`,
			Target: config.RuleTarget{
				Method:  "POST",
				Timeout: 5000000000,
			},
			Body: `{"rule": "a"}`,
		},
		{
			Name:  "rule-b",
			Match: `true`,
			Target: config.RuleTarget{
				Method:  "POST",
				Timeout: 5000000000,
			},
			Body: `{"rule": "b"}`,
		},
	}

	r, _ := setupWebhookHandler(t, rules, target)

	body := []byte(`{"test": true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int32(2), callCount.Load(), "both matching rules should dispatch")
}

func TestWebhookHandler_ConcurrentDispatch(t *testing.T) {
	// Each handler sleeps 200ms; if dispatched concurrently, total < 300ms for 3 rules.
	// If serial, would take >= 600ms.
	var callCount atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	rules := make([]config.Rule, 3)
	for i := range rules {
		rules[i] = config.Rule{
			Name:  "slow-rule-" + string(rune('a'+i)),
			Match: `true`,
			Target: config.RuleTarget{
				Method:  "POST",
				Timeout: 5 * time.Second,
			},
			Body: `{"rule": "test"}`,
		}
	}

	r, _ := setupWebhookHandler(t, rules, target)

	body := []byte(`{"test": true}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	r.ServeHTTP(w, req)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, int32(3), callCount.Load(), "all 3 rules should dispatch")

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(3), resp["matched"])
	assert.Equal(t, float64(3), resp["dispatched"])

	// Concurrent dispatch: 3 x 200ms should complete in ~200ms, not 600ms
	assert.Less(t, elapsed, 500*time.Millisecond,
		"concurrent dispatch should complete in < 500ms, got %v", elapsed)
}

func TestWebhookHandler_AlertmanagerToLark(t *testing.T) {
	var receivedBody map[string]any
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	rules := []config.Rule{
		{
			Name:  "alertmanager-to-lark",
			Match: `len(payload.alerts) > 0`,
			Target: config.RuleTarget{
				Method:  "POST",
				Timeout: 5000000000,
			},
			Body: `{
				"msg_type": "interactive",
				"card": {
					"header": {
						"title": {
							"content": (payload.alerts[0].status == "firing" ? "🔥 " : "✅ ") + payload.alerts[0].labels.alertname,
							"tag": "plain_text"
						}
					}
				}
			}`,
		},
	}

	r, _ := setupWebhookHandler(t, rules, target)

	payload := map[string]any{
		"alerts": []any{
			map[string]any{
				"status": "firing",
				"labels": map[string]any{
					"alertname":   "HighMemoryUsage",
					"lark_bot_id": "abc-123",
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "interactive", receivedBody["msg_type"])

	card := receivedBody["card"].(map[string]any)
	header := card["header"].(map[string]any)
	title := header["title"].(map[string]any)
	assert.Equal(t, "🔥 HighMemoryUsage", title["content"])
}
