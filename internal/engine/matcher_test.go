package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileMatchExpr_Valid(t *testing.T) {
	compiled, err := CompileMatchExpr(`len(payload.alerts) > 0`)
	require.NoError(t, err)
	assert.NotNil(t, compiled)
}

func TestCompileMatchExpr_Invalid(t *testing.T) {
	_, err := CompileMatchExpr(`this is not valid !!!`)
	assert.Error(t, err)
}

func TestMatchRule_True(t *testing.T) {
	compiled, err := CompileMatchExpr(`len(payload.alerts) > 0`)
	require.NoError(t, err)

	payload := map[string]any{
		"alerts": []any{
			map[string]any{"status": "firing"},
		},
	}

	matched, err := MatchRule(compiled, payload)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchRule_False(t *testing.T) {
	compiled, err := CompileMatchExpr(`len(payload.alerts) > 0`)
	require.NoError(t, err)

	payload := map[string]any{
		"alerts": []any{},
	}

	matched, err := MatchRule(compiled, payload)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestMatchRule_StringComparison(t *testing.T) {
	compiled, err := CompileMatchExpr(`payload.status == "firing"`)
	require.NoError(t, err)

	payload := map[string]any{
		"status": "firing",
	}

	matched, err := MatchRule(compiled, payload)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchRule_NestedField(t *testing.T) {
	compiled, err := CompileMatchExpr(`payload.alerts[0].labels.severity == "critical"`)
	require.NoError(t, err)

	payload := map[string]any{
		"alerts": []any{
			map[string]any{
				"labels": map[string]any{
					"severity": "critical",
				},
			},
		},
	}

	matched, err := MatchRule(compiled, payload)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchRule_WithCustomFunction(t *testing.T) {
	compiled, err := CompileMatchExpr(`lower(payload.status) == "firing"`)
	require.NoError(t, err)

	payload := map[string]any{
		"status": "FIRING",
	}

	matched, err := MatchRule(compiled, payload)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestMatchRule_NonBoolResult(t *testing.T) {
	compiled, err := CompileMatchExpr(`payload.status`)
	require.NoError(t, err)

	payload := map[string]any{
		"status": "firing",
	}

	_, err = MatchRule(compiled, payload)
	assert.Error(t, err, "non-bool result should return error")
}
