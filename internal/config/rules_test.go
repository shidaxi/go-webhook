package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRulesFromFile_Valid(t *testing.T) {
	// Use the project's sample rules.yaml
	rules, err := LoadRulesFromFile("../../configs/rules.yaml")
	require.NoError(t, err)
	require.Len(t, rules, 1)

	r := rules[0]
	assert.Equal(t, "alertmanager-to-lark", r.Name)
	assert.Equal(t, `len(payload.alerts) > 0`, r.Match)
	assert.Equal(t, "POST", r.Target.Method)
	assert.Contains(t, r.Target.URL, "lark_bot_id")
	assert.NotEmpty(t, r.Body)
}

func TestLoadRulesFromFile_MultipleRules(t *testing.T) {
	content := `rules:
  - name: rule-one
    match: 'payload.type == "a"'
    target:
      url: '"https://example.com/a"'
      method: POST
    body: '{"type": "a"}'
  - name: rule-two
    match: 'payload.type == "b"'
    target:
      url: '"https://example.com/b"'
      method: POST
      timeout: 5s
    body: '{"type": "b"}'
`
	tmpFile := writeTemp(t, content)

	rules, err := LoadRulesFromFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, rules, 2)
	assert.Equal(t, "rule-one", rules[0].Name)
	assert.Equal(t, "rule-two", rules[1].Name)
}

func TestLoadRulesFromFile_NotFound(t *testing.T) {
	_, err := LoadRulesFromFile("/nonexistent/rules.yaml")
	assert.Error(t, err)
}

func TestLoadRulesFromFile_InvalidYAML(t *testing.T) {
	tmpFile := writeTemp(t, `not: valid: yaml: [`)
	_, err := LoadRulesFromFile(tmpFile)
	assert.Error(t, err)
}

func TestLoadRulesFromFile_EmptyRules(t *testing.T) {
	tmpFile := writeTemp(t, `rules: []`)
	rules, err := LoadRulesFromFile(tmpFile)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestLoadRulesFromFile_DefaultMethod(t *testing.T) {
	content := `rules:
  - name: no-method
    match: 'true'
    target:
      url: '"https://example.com"'
    body: '{}'
`
	tmpFile := writeTemp(t, content)

	rules, err := LoadRulesFromFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "POST", rules[0].Target.Method, "default method should be POST")
}

func TestLoadRulesFromFile_DefaultTimeout(t *testing.T) {
	content := `rules:
  - name: no-timeout
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{}'
`
	tmpFile := writeTemp(t, content)

	rules, err := LoadRulesFromFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, DefaultTimeout, rules[0].Target.Timeout, "default timeout should be 10s")
}

// writeTemp creates a temp file with the given content and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}
