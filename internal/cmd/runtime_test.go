package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// Smoke tests for the runtime adapters in runtime.go. These exercise
// the real OS wrappers against portable programs (cat, true, false)
// so we know the thin glue actually works — fakes alone don't catch
// "forgot to set Stdin" / "forgot to capture exit code" mistakes.
//
// Pattern matches stdlib os/exec_test.go (which runs /bin/echo etc.).
// Skipped on Windows where these programs aren't guaranteed.

func skipIfWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("portable Unix utilities (cat/true/false) unavailable on Windows")
	}
}

func TestRealExecRunner_Run_CapturesStdoutAndExitZero(t *testing.T) {
	skipIfWindows(t)

	r := &execRunner{}
	out, code, err := r.Run("echo", "hello")

	require.NoError(t, err)
	require.Equal(t, 0, code)
	require.Equal(t, "hello\n", out)
}

func TestRealExecRunner_Run_CapturesNonZeroExit(t *testing.T) {
	skipIfWindows(t)

	r := &execRunner{}
	_, code, err := r.Run("false")

	require.NoError(t, err, "non-zero exit must not surface as Go error")
	require.NotEqual(t, 0, code, "exit code must reflect failure")
}

func TestRealExecRunner_RunWithStdin_PipesStdin(t *testing.T) {
	skipIfWindows(t)

	r := &execRunner{}
	const payload = "line one\nline two\n"

	out, code, err := r.RunWithStdin(payload, "cat")

	require.NoError(t, err)
	require.Equal(t, 0, code)
	require.Equal(t, payload, out,
		"cat must echo back the stdin payload byte-for-byte")
}

func TestRealExecRunner_RunWithStdin_CapturesNonZeroExit(t *testing.T) {
	skipIfWindows(t)

	r := &execRunner{}
	_, code, err := r.RunWithStdin("ignored", "false")

	require.NoError(t, err)
	require.NotEqual(t, 0, code)
}

func TestRealExecRunner_RunWithEnv_PassesEnv(t *testing.T) {
	skipIfWindows(t)

	r := &execRunner{}
	out, code, err := r.RunWithEnv(
		map[string]string{"GREETING": "hello-from-env"},
		"sh", "-c", "echo $GREETING",
	)

	require.NoError(t, err)
	require.Equal(t, 0, code)
	require.Equal(t, "hello-from-env\n", out)
}

func TestRealOsFileSystem_FindFiles_RecursivelyMatchesByName(t *testing.T) {
	root := t.TempDir()

	// Create a small tree:
	//   root/a.node
	//   root/sub/b.node
	//   root/sub/c.txt   (must NOT match)
	//   root/sub/deep/d.node
	for _, rel := range []string{"a.node", "sub/b.node", "sub/c.txt", "sub/deep/d.node"} {
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte("x"), 0o644))
	}

	fs := &osFileSystem{}
	matches, err := fs.FindFiles(root, "*.node")
	require.NoError(t, err)
	require.Len(t, matches, 3, "must find all three .node files; got %v", matches)

	// Build a set of basenames for order-independent assertion.
	names := map[string]bool{}
	for _, m := range matches {
		names[filepath.Base(m)] = true
	}
	require.True(t, names["a.node"])
	require.True(t, names["b.node"])
	require.True(t, names["d.node"])
	require.False(t, names["c.txt"], "must not match non-.node files")
}

func TestRealOsFileSystem_FindFiles_ReturnsNothingForMissingRoot(t *testing.T) {
	fs := &osFileSystem{}
	matches, err := fs.FindFiles("/nonexistent/path/that/should/not/exist", "*.node")
	require.NoError(t, err, "missing root must not be an error")
	require.Empty(t, matches)
}
