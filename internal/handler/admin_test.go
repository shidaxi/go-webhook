package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAdminRouter(t *testing.T) (*gin.Engine, *engine.RuleStore) {
	t.Helper()
	store := engine.NewRuleStore()

	cfg := config.AppConfig{
		Server: config.ServerConfig{Port: 8080},
		Admin:  config.AdminConfig{Port: 9090},
		Log:    config.LogConfig{Format: "json"},
		Rules:  config.RulesConfig{Path: "configs/rules.yaml"},
	}

	r := gin.New()
	h := NewAdminHandler(store, cfg)
	r.GET("/admin/healthz", h.Healthz)
	r.GET("/admin/rules", h.Rules)
	r.GET("/admin/config", h.Config)
	return r, store
}

func TestAdminHandler_Healthz(t *testing.T) {
	r, _ := setupAdminRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestAdminHandler_Rules_Empty(t *testing.T) {
	r, _ := setupAdminRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/rules", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	rules := resp["rules"].([]any)
	assert.Empty(t, rules)
}

func TestAdminHandler_Rules_WithRules(t *testing.T) {
	r, store := setupAdminRouter(t)

	compiled := engine.CompileRules([]config.Rule{
		{
			Name:  "test-rule",
			Match: `true`,
			Target: config.RuleTarget{
				URL:    `"https://example.com"`,
				Method: "POST",
			},
			Body: `{"ok": true}`,
		},
	})
	store.SetRules(compiled)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/rules", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	rules := resp["rules"].([]any)
	require.Len(t, rules, 1)

	rule := rules[0].(map[string]any)
	assert.Equal(t, "test-rule", rule["name"])
	assert.Equal(t, true, rule["compiled"])
}

func TestAdminHandler_Config_Redaction(t *testing.T) {
	store := engine.NewRuleStore()

	cfg := config.AppConfig{
		Server: config.ServerConfig{Port: 8080},
		Admin:  config.AdminConfig{Port: 9090},
		Log:    config.LogConfig{Format: "json"},
		Rules:  config.RulesConfig{Path: "configs/rules.yaml"},
	}

	r := gin.New()
	h := NewAdminHandler(store, cfg)
	r.GET("/admin/config", h.Config)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/config", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	server := resp["server"].(map[string]any)
	assert.Equal(t, float64(8080), server["port"])
}
