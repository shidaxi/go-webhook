package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRulesTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestNewRuleStore(t *testing.T) {
	store := NewRuleStore()
	rules := store.GetRules()
	assert.Empty(t, rules)
}

func TestRuleStore_LoadAndCompile(t *testing.T) {
	content := `rules:
  - name: test-rule
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"ok": true}'
`
	tmpFile := writeRulesTemp(t, content)
	store := NewRuleStore()

	err := store.LoadAndCompile(tmpFile)
	require.NoError(t, err)

	rules := store.GetRules()
	require.Len(t, rules, 1)
	assert.Equal(t, "test-rule", rules[0].Rule.Name)
	assert.NoError(t, rules[0].CompileError)
}

func TestRuleStore_LoadAndCompile_KeepsOldOnError(t *testing.T) {
	validContent := `rules:
  - name: valid-rule
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"ok": true}'
`
	tmpFile := writeRulesTemp(t, validContent)
	store := NewRuleStore()
	require.NoError(t, store.LoadAndCompile(tmpFile))

	err := store.LoadAndCompile("/nonexistent/rules.yaml")
	assert.Error(t, err)

	rules := store.GetRules()
	require.Len(t, rules, 1, "should keep old rules on load failure")
	assert.Equal(t, "valid-rule", rules[0].Rule.Name)
}

func TestRuleStore_LoadAndCompile_ReplacesOnSuccess(t *testing.T) {
	content1 := `rules:
  - name: rule-v1
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"v": 1}'
`
	content2 := `rules:
  - name: rule-v2
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"v": 2}'
`
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content1), 0o644))

	store := NewRuleStore()
	require.NoError(t, store.LoadAndCompile(path))
	assert.Equal(t, "rule-v1", store.GetRules()[0].Rule.Name)

	require.NoError(t, os.WriteFile(path, []byte(content2), 0o644))
	require.NoError(t, store.LoadAndCompile(path))
	assert.Equal(t, "rule-v2", store.GetRules()[0].Rule.Name)
}

func TestRuleStore_WatchRules(t *testing.T) {
	content1 := `rules:
  - name: watch-v1
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"v": 1}'
`
	content2 := `rules:
  - name: watch-v2
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"v": 2}'
`
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content1), 0o644))

	store := NewRuleStore()
	require.NoError(t, store.LoadAndCompile(path))

	stop, err := store.WatchRules(path)
	require.NoError(t, err)
	defer stop()

	require.NoError(t, os.WriteFile(path, []byte(content2), 0o644))

	assert.Eventually(t, func() bool {
		rules := store.GetRules()
		return len(rules) > 0 && rules[0].Rule.Name == "watch-v2"
	}, 3*time.Second, 100*time.Millisecond, "rules should reload after file change")
}

func TestRuleStore_GetRules_Snapshot(t *testing.T) {
	store := NewRuleStore()
	content := `rules:
  - name: snap-test
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"ok": true}'
`
	tmpFile := writeRulesTemp(t, content)
	require.NoError(t, store.LoadAndCompile(tmpFile))

	snapshot := store.GetRules()
	require.Len(t, snapshot, 1)

	content2 := `rules:
  - name: snap-test-2
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"ok": true}'
`
	tmpFile2 := writeRulesTemp(t, content2)
	require.NoError(t, store.LoadAndCompile(tmpFile2))

	assert.Equal(t, "snap-test", snapshot[0].Rule.Name)
	assert.Equal(t, "snap-test-2", store.GetRules()[0].Rule.Name)
}

func TestRuleStore_ConcurrentAccess(t *testing.T) {
	store := NewRuleStore()
	content := `rules:
  - name: concurrent-test
    match: 'true'
    target:
      url: '"https://example.com"'
      method: POST
    body: '{"ok": true}'
`
	tmpFile := writeRulesTemp(t, content)
	require.NoError(t, store.LoadAndCompile(tmpFile))

	done := make(chan struct{})
	for range 10 {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = store.GetRules()
				}
			}
		}()
	}
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				_ = store.LoadAndCompile(tmpFile)
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	close(done)
}
