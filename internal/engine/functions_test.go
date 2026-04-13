package engine

import (
	"os"
	"testing"
	"time"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomFunctions_Now(t *testing.T) {
	env := map[string]any{}
	program, err := expr.Compile(`now()`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)

	ts, ok := result.(string)
	require.True(t, ok, "now() should return a string")

	_, err = time.Parse(time.RFC3339, ts)
	assert.NoError(t, err, "now() should return RFC3339 formatted time")
}

func TestCustomFunctions_Env(t *testing.T) {
	t.Setenv("TEST_WEBHOOK_VAR", "hello-world")

	env := map[string]any{}
	program, err := expr.Compile(`env("TEST_WEBHOOK_VAR")`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "hello-world", result)
}

func TestCustomFunctions_EnvMissing(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR_12345")

	env := map[string]any{}
	program, err := expr.Compile(`env("NONEXISTENT_VAR_12345")`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "", result, "env() should return empty string for missing var")
}

func TestCustomFunctions_Lower(t *testing.T) {
	env := map[string]any{}
	program, err := expr.Compile(`lower("HELLO World")`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestCustomFunctions_Upper(t *testing.T) {
	env := map[string]any{}
	program, err := expr.Compile(`upper("hello World")`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "HELLO WORLD", result)
}

func TestCustomFunctions_Join(t *testing.T) {
	env := map[string]any{}
	program, err := expr.Compile(`join(["a", "b", "c"], "-")`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "a-b-c", result)
}

func TestCustomFunctions_Split(t *testing.T) {
	env := map[string]any{}
	program, err := expr.Compile(`split("a-b-c", "-")`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b", "c"}, result)
}

func TestCustomFunctions_ComposedExpression(t *testing.T) {
	env := map[string]any{
		"payload": map[string]any{
			"name": "ALERT_NAME",
		},
	}
	program, err := expr.Compile(`lower(payload.name) + "-" + upper("test")`, ExprOptions(env)...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "alert_name-TEST", result)
}
