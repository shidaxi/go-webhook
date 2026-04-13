//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/shidaxi/go-webhook/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startServers boots both webhook and admin engines on random ports.
// Returns webhookURL, adminURL, and a cleanup function.
func startServers(t *testing.T, rulesPath string) (string, string, func()) {
	t.Helper()

	store := engine.NewRuleStore()
	require.NoError(t, store.LoadAndCompile(rulesPath))

	cfg := config.AppConfig{
		Server: config.ServerConfig{Port: 0},
		Admin:  config.AdminConfig{Port: 0},
		Log:    config.LogConfig{Format: "json"},
		Rules:  config.RulesConfig{Path: rulesPath},
	}

	webhookEngine := server.NewWebhookEngine(store, "")
	adminEngine := server.NewAdminEngine(store, cfg)

	wl, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	al, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	wSrv := &http.Server{Handler: webhookEngine}
	aSrv := &http.Server{Handler: adminEngine}

	go wSrv.Serve(wl)
	go aSrv.Serve(al)

	webhookURL := fmt.Sprintf("http://%s", wl.Addr().String())
	adminURL := fmt.Sprintf("http://%s", al.Addr().String())

	cleanup := func() {
		wSrv.Close()
		aSrv.Close()
	}

	return webhookURL, adminURL, cleanup
}

// writeRulesFile creates a temporary rules.yaml that routes to the mock server.
func writeRulesFile(t *testing.T, mockURL string) string {
	t.Helper()

	rulesContent := fmt.Sprintf(`rules:
  - name: alertmanager-to-mock
    match: 'len(payload.alerts) > 0'
    target:
      url: '"%s/hook/" + payload.alerts[0].labels.lark_bot_id'
      method: POST
      timeout: 5s
      headers:
        Content-Type: application/json
    body: |
      {
        "msg_type": "interactive",
        "card": {
          "header": {
            "title": {
              "content": (payload.alerts[0].status == "firing" ? "🔥 " : "✅ ") + payload.alerts[0].labels.alertname,
              "tag": "plain_text"
            },
            "template": payload.alerts[0].status == "firing" ? "red" : "green"
          },
          "elements": [
            {
              "tag": "div",
              "text": {
                "content": "**Instance:** " + payload.alerts[0].labels.instance + "\n**Severity:** " + payload.alerts[0].labels.severity + "\n**Summary:** " + payload.alerts[0].annotations.summary,
                "tag": "lark_md"
              }
            }
          ]
        }
      }
  - name: no-match-rule
    match: 'payload.status == "never_matches"'
    target:
      url: '"%s/should-not-be-called"'
      method: POST
      timeout: 5s
    body: '{"noop": true}'
`, mockURL, mockURL)

	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(rulesContent), 0644))
	return path
}

func TestE2E_WebhookFiringAlert(t *testing.T) {
	// Arrange: mock Lark bot target that captures the request
	var capturedBody map[string]any
	var capturedPath string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer mock.Close()

	rulesPath := writeRulesFile(t, mock.URL)
	webhookURL, _, cleanup := startServers(t, rulesPath)
	defer cleanup()

	// Act: send a firing alertmanager payload
	payload := loadFixture(t, "alertmanager.json")
	resp, body := postJSON(t, webhookURL+"/webhook", payload)

	// Assert: webhook response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, float64(1), body["matched"])
	assert.Equal(t, float64(1), body["dispatched"])

	// Assert: mock received the correct forwarded request
	assert.Equal(t, "/hook/abc-123-def", capturedPath)
	require.NotNil(t, capturedBody)
	assert.Equal(t, "interactive", capturedBody["msg_type"])

	card, ok := capturedBody["card"].(map[string]any)
	require.True(t, ok)
	header := card["header"].(map[string]any)
	title := header["title"].(map[string]any)
	assert.Equal(t, "🔥 HighMemoryUsage", title["content"])
	assert.Equal(t, "red", header["template"])
}

func TestE2E_WebhookResolvedAlert(t *testing.T) {
	var capturedBody map[string]any
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()

	rulesPath := writeRulesFile(t, mock.URL)
	webhookURL, _, cleanup := startServers(t, rulesPath)
	defer cleanup()

	payload := loadFixture(t, "alertmanager_resolved.json")
	resp, body := postJSON(t, webhookURL+"/webhook", payload)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, float64(1), body["matched"])
	assert.Equal(t, float64(1), body["dispatched"])

	require.NotNil(t, capturedBody)
	card := capturedBody["card"].(map[string]any)
	header := card["header"].(map[string]any)
	title := header["title"].(map[string]any)
	assert.Equal(t, "✅ HighMemoryUsage", title["content"])
	assert.Equal(t, "green", header["template"])
}

func TestE2E_WebhookNoMatchingRules(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("mock should not be called for non-matching payload")
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()

	rulesPath := writeRulesFile(t, mock.URL)
	webhookURL, _, cleanup := startServers(t, rulesPath)
	defer cleanup()

	// Send a payload that doesn't match any rule (empty alerts)
	payload := map[string]any{
		"version": "4",
		"status":  "firing",
		"alerts":  []any{},
	}
	resp, body := postJSON(t, webhookURL+"/webhook", payload)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, float64(0), body["matched"])
	assert.Equal(t, float64(0), body["dispatched"])
}

func TestE2E_WebhookInvalidBody(t *testing.T) {
	rulesPath := writeRulesFile(t, "http://localhost:1")
	webhookURL, _, cleanup := startServers(t, rulesPath)
	defer cleanup()

	resp, err := http.Post(webhookURL+"/webhook", "application/json", bytes.NewBufferString("not json"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "INVALID_BODY", body["code"])
}

func TestE2E_AdminHealthz(t *testing.T) {
	rulesPath := writeRulesFile(t, "http://localhost:1")
	_, adminURL, cleanup := startServers(t, rulesPath)
	defer cleanup()

	resp, err := http.Get(adminURL + "/admin/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "ok", body["status"])
}

func TestE2E_AdminRules(t *testing.T) {
	rulesPath := writeRulesFile(t, "http://localhost:1")
	_, adminURL, cleanup := startServers(t, rulesPath)
	defer cleanup()

	resp, err := http.Get(adminURL + "/admin/rules")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	rules, ok := body["rules"].([]any)
	require.True(t, ok)
	assert.Equal(t, 2, len(rules), "should have 2 rules loaded")

	first := rules[0].(map[string]any)
	assert.Equal(t, "alertmanager-to-mock", first["name"])
	assert.Equal(t, true, first["compiled"])
}

func TestE2E_AdminConfig(t *testing.T) {
	rulesPath := writeRulesFile(t, "http://localhost:1")
	_, adminURL, cleanup := startServers(t, rulesPath)
	defer cleanup()

	resp, err := http.Get(adminURL + "/admin/config")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)

	// Verify config structure
	serverCfg, ok := body["server"].(map[string]any)
	require.True(t, ok)
	assert.NotNil(t, serverCfg["port"])
}

func TestE2E_TargetServerDown(t *testing.T) {
	// Create a server and immediately close it to get a valid but unreachable URL
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := closedServer.URL
	closedServer.Close()

	rulesPath := writeRulesFile(t, closedURL)
	webhookURL, _, cleanup := startServers(t, rulesPath)
	defer cleanup()

	payload := loadFixture(t, "alertmanager.json")

	client := &http.Client{Timeout: 30 * time.Second}
	jsonData, _ := json.Marshal(payload)
	resp, err := client.Post(webhookURL+"/webhook", "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, float64(1), body["matched"])
	assert.Equal(t, float64(0), body["dispatched"], "dispatch should fail when target is down")
}

// --- helpers ---

func loadFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "fixtures", name))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func postJSON(t *testing.T, url string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	jsonData, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	var body map[string]any
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &body))

	return resp, body
}
