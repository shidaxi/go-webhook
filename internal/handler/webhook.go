package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/shidaxi/go-webhook/internal/logger"
	"go.uber.org/zap"
)

const defaultMaxRetries = 3

// WebhookHandler handles incoming webhook requests.
type WebhookHandler struct {
	store *engine.RuleStore
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(store *engine.RuleStore) *WebhookHandler {
	return &WebhookHandler{store: store}
}

// Handle processes an incoming webhook request.
// @Summary      Receive and forward webhook
// @Description  Receives a JSON payload, matches against loaded rules, transforms the body, and dispatches to target URLs.
// @Tags         webhook
// @Accept       json
// @Produce      json
// @Param        body  body  object  true  "Incoming webhook JSON payload"
// @Success      200  {object}  WebhookResponse
// @Failure      400  {object}  ErrorResponse
// @Router       /webhook [post]
func (h *WebhookHandler) Handle(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON body",
			"code":  "INVALID_BODY",
		})
		return
	}

	rules := h.store.GetRules()
	results := h.processRules(c, payload, rules)

	matched := 0
	dispatched := 0
	for _, r := range results {
		matched++
		if r.Success {
			dispatched++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"matched":    matched,
		"dispatched": dispatched,
		"results":    results,
	})
}

func (h *WebhookHandler) processRules(c *gin.Context, payload map[string]any, rules []engine.CompiledRule) []config.DispatchResult {
	var results []config.DispatchResult

	for _, cr := range rules {
		if cr.CompileError != nil {
			continue
		}

		matched, err := engine.MatchRule(cr.MatchProgram, payload)
		if err != nil {
			logger.L().Warn("match evaluation failed",
				zap.String("rule", cr.Rule.Name),
				zap.Error(err),
			)
			continue
		}

		if !matched {
			continue
		}

		targetURL, err := engine.TransformURL(cr.URLProgram, payload)
		if err != nil {
			logger.L().Error("URL transform failed",
				zap.String("rule", cr.Rule.Name),
				zap.Error(err),
			)
			continue
		}

		body, err := engine.TransformBody(cr.BodyProgram, payload)
		if err != nil {
			logger.L().Error("body transform failed",
				zap.String("rule", cr.Rule.Name),
				zap.Error(err),
			)
			continue
		}

		result := engine.Dispatch(
			c.Request.Context(),
			targetURL,
			cr.Rule.Target.Method,
			body,
			cr.Rule.Target.Headers,
			cr.Rule.Target.Timeout,
			defaultMaxRetries,
		)
		result.RuleName = cr.Rule.Name

		if result.Error != nil {
			logger.L().Error("dispatch failed",
				zap.String("rule", cr.Rule.Name),
				zap.String("target", targetURL),
				zap.Error(result.Error),
			)
		} else {
			logger.L().Info("dispatch success",
				zap.String("rule", cr.Rule.Name),
				zap.String("target", targetURL),
				zap.Int("status", result.StatusCode),
			)
		}

		results = append(results, result)
	}

	return results
}
