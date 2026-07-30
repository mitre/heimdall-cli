package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakePrompter_Input(t *testing.T) {
	p := &FakePrompter{
		Inputs: map[string]string{
			"Database host": "db.example.com",
		},
	}
	val, err := p.Input("Database host", "localhost")
	require.NoError(t, err)
	assert.Equal(t, "db.example.com", val)
}

func TestFakePrompter_Input_UsesDefault(t *testing.T) {
	p := &FakePrompter{
		Inputs: map[string]string{},
	}
	val, err := p.Input("Database host", "localhost")
	require.NoError(t, err)
	assert.Equal(t, "localhost", val)
}

func TestFakePrompter_Password(t *testing.T) {
	p := &FakePrompter{
		Inputs: map[string]string{
			"Database password": "secret123",
		},
	}
	val, err := p.Password("Database password")
	require.NoError(t, err)
	assert.Equal(t, "secret123", val)
}

func TestFakePrompter_Confirm(t *testing.T) {
	p := &FakePrompter{
		Confirms: map[string]bool{
			"Continue?": true,
		},
	}
	val, err := p.Confirm("Continue?", false)
	require.NoError(t, err)
	assert.True(t, val)
}

func TestFakePrompter_Confirm_UsesDefault(t *testing.T) {
	p := &FakePrompter{
		Confirms: map[string]bool{},
	}
	val, err := p.Confirm("Continue?", false)
	require.NoError(t, err)
	assert.False(t, val)
}

func TestFakePrompter_Select(t *testing.T) {
	p := &FakePrompter{
		Selects: map[string]int{
			"Choose option": 2,
		},
	}
	val, err := p.Select("Choose option", "", []string{"a", "b", "c"})
	require.NoError(t, err)
	assert.Equal(t, 2, val)
}

func TestFakePrompter_CanPrompt(t *testing.T) {
	p := &FakePrompter{IsTTY: true}
	assert.True(t, p.CanPrompt())

	p.IsTTY = false
	assert.False(t, p.CanPrompt())
}

func TestFakeTerminalDetector(t *testing.T) {
	d := &FakeTerminalDetector{TTY: true}
	assert.True(t, d.IsTerminal())

	d.TTY = false
	assert.False(t, d.IsTerminal())
}
