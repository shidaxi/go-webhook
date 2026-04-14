package handler

import (
	"net/http"
	"sync"

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
	matched, results := h.processRules(c, payload, rules)

	dispatched := 0
	for _, r := range results {
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

// dispatchJob holds all info needed to dispatch a single HTTP request.
type dispatchJob struct {
	ruleName  string
	targetURL string
	method    string
	body      map[string]any
	rule      config.Rule
}

func (h *WebhookHandler) processRules(c *gin.Context, payload map[string]any, rules []engine.CompiledRule) (int, []config.DispatchResult) {
	// Phase 1: match, expand forEach, and transform (sequential)
	var jobs []dispatchJob
	matched := 0

	for _, cr := range rules {
		if cr.CompileError != nil {
			continue
		}

		ok, err := engine.MatchRule(cr.MatchProgram, payload)
		if err != nil {
			logger.L().Warn("match evaluation failed",
				zap.String("rule", cr.Rule.Name),
				zap.Error(err),
			)
			continue
		}

		if !ok {
			continue
		}

		matched++

		ruleJobs := h.expandRule(cr, payload)
		jobs = append(jobs, ruleJobs...)
	}

	if len(jobs) == 0 {
		return matched, nil
	}

	// Phase 2: dispatch concurrently
	results := make([]config.DispatchResult, len(jobs))
	var wg sync.WaitGroup
	wg.Add(len(jobs))

	for i, job := range jobs {
		go func(idx int, j dispatchJob) {
			defer wg.Done()

			result := engine.Dispatch(
				c.Request.Context(),
				j.targetURL,
				j.method,
				j.body,
				j.rule.Target.Headers,
				j.rule.Target.Timeout,
				defaultMaxRetries,
			)
			result.RuleName = j.ruleName

			if result.Error != nil {
				logger.L().Error("dispatch failed",
					zap.String("rule", j.ruleName),
					zap.String("target", j.targetURL),
					zap.Error(result.Error),
				)
			} else {
				logger.L().Info("dispatch success",
					zap.String("rule", j.ruleName),
					zap.String("target", j.targetURL),
					zap.Int("status", result.StatusCode),
				)
			}

			results[idx] = result
		}(i, job)
	}

	wg.Wait()
	return matched, results
}

// expandRule expands a single matched rule into one or more dispatch jobs.
// If the rule has a forEach expression, it evaluates it and creates a job per item.
// Otherwise, it creates a single job.
func (h *WebhookHandler) expandRule(cr engine.CompiledRule, payload map[string]any) []dispatchJob {
	if cr.ForEachProgram == nil {
		return h.buildJob(cr, payload, nil)
	}

	items, err := engine.EvalForEach(cr.ForEachProgram, payload)
	if err != nil {
		logger.L().Error("forEach evaluation failed",
			zap.String("rule", cr.Rule.Name),
			zap.Error(err),
		)
		return nil
	}

	var jobs []dispatchJob
	for _, item := range items {
		itemJobs := h.buildJob(cr, payload, item)
		jobs = append(jobs, itemJobs...)
	}
	return jobs
}

// buildJob transforms a single rule (with optional item) into dispatch jobs.
func (h *WebhookHandler) buildJob(cr engine.CompiledRule, payload map[string]any, item any) []dispatchJob {
	targetURL, err := engine.TransformURLWithItem(cr.URLProgram, payload, item)
	if err != nil {
		logger.L().Error("URL transform failed",
			zap.String("rule", cr.Rule.Name),
			zap.Error(err),
		)
		return nil
	}

	body, err := engine.TransformBodyWithItem(cr.BodyProgram, payload, item)
	if err != nil {
		logger.L().Error("body transform failed",
			zap.String("rule", cr.Rule.Name),
			zap.Error(err),
		)
		return nil
	}

	return []dispatchJob{{
		ruleName:  cr.Rule.Name,
		targetURL: targetURL,
		method:    cr.Rule.Target.Method,
		body:      body,
		rule:      cr.Rule,
	}}
}
