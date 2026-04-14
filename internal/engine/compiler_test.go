package engine

import (
	"testing"
	"time"

	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileRules_AllValid(t *testing.T) {
	rules := []config.Rule{
		{
			Name:  "test-rule",
			Match: `len(payload.alerts) > 0`,
			Target: config.RuleTarget{
				URL:     `"https://example.com/hook/" + payload.alerts[0].labels.bot_id`,
				Method:  "POST",
				Timeout: 10 * time.Second,
			},
			Body: `{"msg": payload.alerts[0].labels.alertname}`,
		},
	}

	compiled := CompileRules(rules)
	require.Len(t, compiled, 1)

	cr := compiled[0]
	assert.Equal(t, "test-rule", cr.Rule.Name)
	assert.NotNil(t, cr.MatchProgram)
	assert.NotNil(t, cr.URLProgram)
	assert.NotNil(t, cr.BodyProgram)
	assert.NoError(t, cr.CompileError)
}

func TestCompileRules_InvalidMatch(t *testing.T) {
	rules := []config.Rule{
		{
			Name:  "bad-match",
			Match: `not valid !!!`,
			Target: config.RuleTarget{
				URL:    `"https://example.com"`,
				Method: "POST",
			},
			Body: `{"ok": true}`,
		},
	}

	compiled := CompileRules(rules)
	require.Len(t, compiled, 1)
	assert.Error(t, compiled[0].CompileError)
	assert.Nil(t, compiled[0].MatchProgram)
}

func TestCompileRules_InvalidURL(t *testing.T) {
	rules := []config.Rule{
		{
			Name:  "bad-url",
			Match: `true`,
			Target: config.RuleTarget{
				URL:    `not valid !!!`,
				Method: "POST",
			},
			Body: `{"ok": true}`,
		},
	}

	compiled := CompileRules(rules)
	require.Len(t, compiled, 1)
	assert.Error(t, compiled[0].CompileError)
}

func TestCompileRules_InvalidBody(t *testing.T) {
	rules := []config.Rule{
		{
			Name:  "bad-body",
			Match: `true`,
			Target: config.RuleTarget{
				URL:    `"https://example.com"`,
				Method: "POST",
			},
			Body: `not valid !!!`,
		},
	}

	compiled := CompileRules(rules)
	require.Len(t, compiled, 1)
	assert.Error(t, compiled[0].CompileError)
}

func TestCompileRules_MixedValidInvalid(t *testing.T) {
	rules := []config.Rule{
		{
			Name:  "valid",
			Match: `true`,
			Target: config.RuleTarget{
				URL:    `"https://example.com"`,
				Method: "POST",
			},
			Body: `{"ok": true}`,
		},
		{
			Name:  "invalid",
			Match: `broken !!!`,
			Target: config.RuleTarget{
				URL:    `"https://example.com"`,
				Method: "POST",
			},
			Body: `{"ok": true}`,
		},
	}

	compiled := CompileRules(rules)
	require.Len(t, compiled, 2)
	assert.NoError(t, compiled[0].CompileError, "first rule should compile fine")
	assert.Error(t, compiled[1].CompileError, "second rule should fail")
}

func TestCompileRules_WithForEach(t *testing.T) {
	rules := []config.Rule{
		{
			Name:    "fan-out",
			Match:   `true`,
			ForEach: `split(payload.ids, ",")`,
			Target: config.RuleTarget{
				URL:    `"https://example.com/" + item`,
				Method: "POST",
			},
			Body: `{"id": item}`,
		},
	}

	compiled := CompileRules(rules)
	require.Len(t, compiled, 1)
	assert.NoError(t, compiled[0].CompileError)
	assert.NotNil(t, compiled[0].ForEachProgram)
	assert.NotNil(t, compiled[0].URLProgram)
	assert.NotNil(t, compiled[0].BodyProgram)
}

func TestCompileRules_InvalidForEach(t *testing.T) {
	rules := []config.Rule{
		{
			Name:    "bad-foreach",
			Match:   `true`,
			ForEach: `broken !!!`,
			Target: config.RuleTarget{
				URL:    `"https://example.com"`,
				Method: "POST",
			},
			Body: `{"ok": true}`,
		},
	}

	compiled := CompileRules(rules)
	require.Len(t, compiled, 1)
	assert.Error(t, compiled[0].CompileError)
	assert.Contains(t, compiled[0].CompileError.Error(), "forEach")
}

func TestCompileRules_Empty(t *testing.T) {
	compiled := CompileRules(nil)
	assert.Empty(t, compiled)
}
