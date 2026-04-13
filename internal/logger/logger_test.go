package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_JSON(t *testing.T) {
	err := Init("json")
	require.NoError(t, err)
	assert.NotNil(t, L())
}

func TestInit_Text(t *testing.T) {
	err := Init("text")
	require.NoError(t, err)
	assert.NotNil(t, L())
}

func TestInit_Empty(t *testing.T) {
	err := Init("")
	require.NoError(t, err, "empty string should default to json")
}

func TestInit_Unknown(t *testing.T) {
	err := Init("xml")
	assert.Error(t, err)
}

func TestL_BeforeInit(t *testing.T) {
	old := global
	global = nil
	defer func() { global = old }()

	l := L()
	assert.NotNil(t, l, "should return nop logger before Init")
}
