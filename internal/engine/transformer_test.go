package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileExpr_Valid(t *testing.T) {
	compiled, err := CompileExpr(`"https://example.com/" + payload.id`)
	require.NoError(t, err)
	assert.NotNil(t, compiled)
}

func TestCompileExpr_Invalid(t *testing.T) {
	_, err := CompileExpr(`not valid !!!`)
	assert.Error(t, err)
}

func TestTransformURL_Static(t *testing.T) {
	compiled, err := CompileExpr(`"https://example.com/webhook"`)
	require.NoError(t, err)

	url, err := TransformURL(compiled, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/webhook", url)
}

func TestTransformURL_DynamicFromPayload(t *testing.T) {
	compiled, err := CompileExpr(`"https://open.larksuite.com/open-apis/bot/v2/hook/" + payload.alerts[0].labels.lark_bot_id`)
	require.NoError(t, err)

	payload := map[string]any{
		"alerts": []any{
			map[string]any{
				"labels": map[string]any{
					"lark_bot_id": "abc-123-def",
				},
			},
		},
	}

	url, err := TransformURL(compiled, payload)
	require.NoError(t, err)
	assert.Equal(t, "https://open.larksuite.com/open-apis/bot/v2/hook/abc-123-def", url)
}

func TestTransformURL_NonStringResult(t *testing.T) {
	compiled, err := CompileExpr(`42`)
	require.NoError(t, err)

	_, err = TransformURL(compiled, map[string]any{})
	assert.Error(t, err, "non-string URL should return error")
}

func TestTransformBody_SimpleMap(t *testing.T) {
	compiled, err := CompileExpr(`{"msg_type": "text", "content": payload.message}`)
	require.NoError(t, err)

	payload := map[string]any{
		"message": "hello world",
	}

	body, err := TransformBody(compiled, payload)
	require.NoError(t, err)
	assert.Equal(t, "text", body["msg_type"])
	assert.Equal(t, "hello world", body["content"])
}

func TestTransformBody_NestedMap(t *testing.T) {
	expr := `{
		"msg_type": "interactive",
		"card": {
			"header": {
				"title": {"content": payload.alerts[0].labels.alertname, "tag": "plain_text"}
			}
		}
	}`
	compiled, err := CompileExpr(expr)
	require.NoError(t, err)

	payload := map[string]any{
		"alerts": []any{
			map[string]any{
				"labels": map[string]any{
					"alertname": "HighMemoryUsage",
				},
			},
		},
	}

	body, err := TransformBody(compiled, payload)
	require.NoError(t, err)

	card, ok := body["card"].(map[string]any)
	require.True(t, ok)
	header, ok := card["header"].(map[string]any)
	require.True(t, ok)
	title, ok := header["title"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "HighMemoryUsage", title["content"])
	assert.Equal(t, "plain_text", title["tag"])
}

func TestTransformBody_ConditionalEmoji(t *testing.T) {
	exprStr := `{
		"title": (payload.alerts[0].status == "firing" ? "🔥 " : "✅ ") + payload.alerts[0].labels.alertname
	}`
	compiled, err := CompileExpr(exprStr)
	require.NoError(t, err)

	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"firing", "firing", "🔥 HighMemoryUsage"},
		{"resolved", "resolved", "✅ HighMemoryUsage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]any{
				"alerts": []any{
					map[string]any{
						"status": tt.status,
						"labels": map[string]any{
							"alertname": "HighMemoryUsage",
						},
					},
				},
			}

			body, err := TransformBody(compiled, payload)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, body["title"])
		})
	}
}

func TestTransformBody_WithCustomFunctions(t *testing.T) {
	compiled, err := CompileExpr(`{"name": lower(payload.NAME), "tags": join(["a","b"], ",")}`)
	require.NoError(t, err)

	payload := map[string]any{"NAME": "HELLO"}

	body, err := TransformBody(compiled, payload)
	require.NoError(t, err)
	assert.Equal(t, "hello", body["name"])
	assert.Equal(t, "a,b", body["tags"])
}

func TestTransformBody_NonMapResult(t *testing.T) {
	compiled, err := CompileExpr(`"just a string"`)
	require.NoError(t, err)

	_, err = TransformBody(compiled, map[string]any{})
	assert.Error(t, err, "non-map body result should return error")
}
