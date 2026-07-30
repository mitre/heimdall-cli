package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSetPortRunner() (*SetPortRunner, *FakeExecRunner, *FakeEnvManager, *FakeSystemdRunner, *bytes.Buffer) {
	out := new(bytes.Buffer)
	exec := &FakeExecRunner{Results: map[string]FakeExecResult{}}
	env := &FakeEnvManager{Env: map[string]string{"PORT": "3000"}}
	systemd := &FakeSystemdRunner{ActiveServices: map[string]bool{}, Properties: map[string]string{}}
	return &SetPortRunner{
		Exec:    exec,
		Env:     env,
		Systemd: systemd,
		Out:     out,
		ErrOut:  new(bytes.Buffer),
	}, exec, env, systemd, out
}

func TestSetPort_InvalidPortZero(t *testing.T) {
	runner, _, _, _, _ := newSetPortRunner()
	err := runner.Run("0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be 1-65535")
}

func TestSetPort_InvalidPort65536(t *testing.T) {
	runner, _, _, _, _ := newSetPortRunner()
	err := runner.Run("65536")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be 1-65535")
}

func TestSetPort_InvalidPortNotNumber(t *testing.T) {
	runner, _, _, _, _ := newSetPortRunner()
	err := runner.Run("abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be 1-65535")
}

func TestSetPort_UpdatesEnvFile(t *testing.T) {
	runner, _, env, _, out := newSetPortRunner()
	err := runner.Run("8080")
	require.NoError(t, err)
	assert.Equal(t, "8080", env.Env["PORT"])
	// Verify output shows old and new port
	assert.Contains(t, out.String(), "PORT=8080 (was 3000)")
}

func TestSetPort_CallsSemanageAndFirewall(t *testing.T) {
	runner, exec, _, _, _ := newSetPortRunner()
	err := runner.Run("8080")
	require.NoError(t, err)

	foundSemanage := false
	foundFirewall := false
	for _, call := range exec.Calls {
		if call.Name == "semanage" {
			foundSemanage = true
			assert.Contains(t, call.Args, "heimdall_server_port_t", "semanage should use heimdall_server_port_t")
			assert.Contains(t, call.Args, "8080", "semanage should use the new port")
		}
		if call.Name == "firewall-cmd" && len(call.Args) > 0 {
			for _, arg := range call.Args {
				if arg == "--add-port=8080/tcp" {
					foundFirewall = true
				}
			}
		}
	}
	assert.True(t, foundSemanage, "should call semanage")
	assert.True(t, foundFirewall, "should call firewall-cmd --add-port=8080/tcp")
}

func TestSetPort_RestartsService(t *testing.T) {
	runner, _, _, systemd, _ := newSetPortRunner()
	err := runner.Run("8080")
	require.NoError(t, err)
	assert.Contains(t, systemd.Actions, "restart:"+ServiceName)
}
