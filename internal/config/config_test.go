package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitConfig_Defaults(t *testing.T) {
	// Init with no config file — should use defaults
	cfg, err := InitConfig("")
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 9090, cfg.Admin.Port)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "configs/rules.yaml", cfg.Rules.Path)
}

func TestInitConfig_FromFile(t *testing.T) {
	content := `
server:
  port: 3000
admin:
  port: 3001
log:
  format: text
rules:
  path: /tmp/rules.yaml
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := InitConfig(path)
	require.NoError(t, err)

	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Equal(t, 3001, cfg.Admin.Port)
	assert.Equal(t, "text", cfg.Log.Format)
	assert.Equal(t, "/tmp/rules.yaml", cfg.Rules.Path)
}

func TestInitConfig_EnvOverride(t *testing.T) {
	t.Setenv("GOWEBHOOK_SERVER_PORT", "9999")

	cfg, err := InitConfig("")
	require.NoError(t, err)

	assert.Equal(t, 9999, cfg.Server.Port)
}

func TestInitConfig_PartialFile(t *testing.T) {
	content := `
server:
  port: 5555
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cfg, err := InitConfig(path)
	require.NoError(t, err)

	assert.Equal(t, 5555, cfg.Server.Port)
	assert.Equal(t, 9090, cfg.Admin.Port, "unset fields should use defaults")
	assert.Equal(t, "json", cfg.Log.Format, "unset fields should use defaults")
}
