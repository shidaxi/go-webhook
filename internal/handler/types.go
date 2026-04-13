package handler

// WebhookResponse is the response from the webhook endpoint.
type WebhookResponse struct {
	Matched    int              `json:"matched" example:"1"`
	Dispatched int              `json:"dispatched" example:"1"`
	Results    []DispatchResult `json:"results"`
}

// DispatchResult describes the outcome of dispatching to one target.
type DispatchResult struct {
	RuleName   string `json:"rule_name" example:"alertmanager-to-lark"`
	TargetURL  string `json:"target_url" example:"https://open.larksuite.com/open-apis/bot/v2/hook/abc"`
	StatusCode int    `json:"status_code" example:"200"`
	Success    bool   `json:"success" example:"true"`
	Error      string `json:"error,omitempty" example:""`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid JSON body"`
	Code  string `json:"code" example:"INVALID_BODY"`
}

// HealthResponse is the health check response.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// RulesResponse lists loaded rules.
type RulesResponse struct {
	Rules []RuleEntry `json:"rules"`
}

// RuleEntry describes one loaded rule.
type RuleEntry struct {
	Name     string `json:"name" example:"alertmanager-to-lark"`
	Match    string `json:"match" example:"len(payload.alerts) > 0"`
	Target   string `json:"target" example:"https://example.com"`
	Method   string `json:"method" example:"POST"`
	Compiled bool   `json:"compiled" example:"true"`
	Error    string `json:"error,omitempty"`
}
