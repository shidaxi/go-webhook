package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
)

// AdminHandler handles admin API requests.
type AdminHandler struct {
	store *engine.RuleStore
	cfg   config.AppConfig
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(store *engine.RuleStore, cfg config.AppConfig) *AdminHandler {
	return &AdminHandler{store: store, cfg: cfg}
}

// Healthz returns a simple health check response.
// @Summary      Health check
// @Description  Returns ok if the service is running.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /admin/healthz [get]
func (h *AdminHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Rules returns the current loaded rules with compile status.
// @Summary      List loaded rules
// @Description  Returns all currently loaded webhook forwarding rules with compile status.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  RulesResponse
// @Router       /admin/rules [get]
func (h *AdminHandler) Rules(c *gin.Context) {
	compiled := h.store.GetRules()
	rules := make([]gin.H, 0, len(compiled))

	for _, cr := range compiled {
		entry := gin.H{
			"name":     cr.Rule.Name,
			"match":    cr.Rule.Match,
			"target":   cr.Rule.Target.URL,
			"method":   cr.Rule.Target.Method,
			"compiled": cr.CompileError == nil,
		}
		if cr.CompileError != nil {
			entry["error"] = cr.CompileError.Error()
		}
		rules = append(rules, entry)
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// Config returns the runtime configuration with sensitive fields redacted.
// @Summary      Runtime configuration
// @Description  Returns the runtime configuration as JSON. Sensitive fields (token, secret, password, key) are redacted.
// @Tags         admin
// @Produce      json
// @Success      200  {object}  object
// @Router       /admin/config [get]
func (h *AdminHandler) Config(c *gin.Context) {
	// Marshal then unmarshal to get a generic map for redaction
	data, _ := json.Marshal(h.cfg)
	var cfgMap map[string]any
	json.Unmarshal(data, &cfgMap)

	redactSensitive(cfgMap)

	c.JSON(http.StatusOK, cfgMap)
}

// redactSensitive replaces values of keys containing sensitive words.
var sensitiveWords = []string{"token", "secret", "password", "key"}

func redactSensitive(m map[string]any) {
	for k, v := range m {
		lower := strings.ToLower(k)
		for _, word := range sensitiveWords {
			if strings.Contains(lower, word) {
				m[k] = "******"
				break
			}
		}
		if nested, ok := v.(map[string]any); ok {
			redactSensitive(nested)
		}
	}
}
