package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "simple key=value",
			input:    "PORT=3000\nHOST=localhost",
			expected: map[string]string{"PORT": "3000", "HOST": "localhost"},
		},
		{
			name:     "quoted value",
			input:    `DATABASE_NAME="heimdall-server-production"`,
			expected: map[string]string{"DATABASE_NAME": "heimdall-server-production"},
		},
		{
			name:     "single-quoted value",
			input:    `SECRET='my-secret'`,
			expected: map[string]string{"SECRET": "my-secret"},
		},
		{
			name:     "comments and blank lines",
			input:    "# This is a comment\n\nPORT=3000\n# Another comment\nHOST=localhost\n",
			expected: map[string]string{"PORT": "3000", "HOST": "localhost"},
		},
		{
			name:     "no equals sign skipped",
			input:    "INVALID_LINE\nPORT=3000",
			expected: map[string]string{"PORT": "3000"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "value with equals sign",
			input:    "URL=postgres://user:pass@host:5432/db?sslmode=require",
			expected: map[string]string{"URL": "postgres://user:pass@host:5432/db?sslmode=require"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseEnv(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileEnvManager_WriteEnvKey(t *testing.T) {
	t.Run("updates existing key", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")
		require.NoError(t, os.WriteFile(path, []byte("PORT=3000\nHOST=localhost\n"), 0640))

		m := &FileEnvManager{Path: path}
		require.NoError(t, m.WriteEnvKey("PORT", "8080"))

		env, err := m.ReadEnv()
		require.NoError(t, err)
		assert.Equal(t, "8080", env["PORT"])
		assert.Equal(t, "localhost", env["HOST"])
	})

	t.Run("adds new key", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")
		require.NoError(t, os.WriteFile(path, []byte("PORT=3000\n"), 0640))

		m := &FileEnvManager{Path: path}
		require.NoError(t, m.WriteEnvKey("HOST", "0.0.0.0"))

		env, err := m.ReadEnv()
		require.NoError(t, err)
		assert.Equal(t, "3000", env["PORT"])
		assert.Equal(t, "0.0.0.0", env["HOST"])
	})

	t.Run("preserves comments", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")
		content := "# Database config\nPORT=3000\n# End\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0640))

		m := &FileEnvManager{Path: path}
		require.NoError(t, m.WriteEnvKey("PORT", "8080"))

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "# Database config")
		assert.Contains(t, string(data), "PORT=8080")
		assert.Contains(t, string(data), "# End")
	})
}

func TestFileEnvManager_ReadEnv(t *testing.T) {
	t.Run("returns empty map for missing file", func(t *testing.T) {
		m := &FileEnvManager{Path: "/nonexistent/path/test.env"}
		env, err := m.ReadEnv()
		// Error is returned but map is always non-nil (safe to access)
		assert.Error(t, err)
		assert.NotNil(t, env)
		assert.Empty(t, env)
	})
}

func TestFileEnvManager_GetEnvFilePath(t *testing.T) {
	t.Run("returns compile-time default", func(t *testing.T) {
		m := NewFileEnvManager()
		assert.Equal(t, EnvFile, m.GetEnvFilePath())
	})

	t.Run("returns custom path when set directly", func(t *testing.T) {
		m := &FileEnvManager{Path: "/tmp/override.env"}
		assert.Equal(t, "/tmp/override.env", m.GetEnvFilePath())
	})
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "(empty)"},
		{"short", "abc", "***"},
		{"exactly 6", "abcdef", "***"},
		{"normal", "mysecretvalue", "mys***lue"},
		{"7 chars", "abcdefg", "abc***efg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskSecret(tt.input))
		})
	}
}

func TestReadEnv_MissingFile(t *testing.T) {
	m := &FileEnvManager{Path: "/nonexistent/path/backend.env"}
	env, err := m.ReadEnv()
	// Error is returned but map is always non-nil (safe to access)
	assert.Error(t, err)
	assert.NotNil(t, env)
	assert.Empty(t, env)
}

func TestFileEnvManager_WriteEnvFile(t *testing.T) {
	t.Run("writes sorted keys to file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")

		m := &FileEnvManager{Path: path}
		entries := map[string]string{
			"ZZZ_LAST":  "last",
			"AAA_FIRST": "first",
			"MMM_MID":   "middle",
		}
		require.NoError(t, m.WriteEnvFile(entries))

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		content := string(data)

		// Keys should be in sorted order
		assert.Equal(t, "AAA_FIRST=first\nMMM_MID=middle\nZZZ_LAST=last\n", content)
	})

	t.Run("replaces existing file content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")
		require.NoError(t, os.WriteFile(path, []byte("OLD_KEY=old_value\n"), 0640))

		m := &FileEnvManager{Path: path}
		require.NoError(t, m.WriteEnvFile(map[string]string{
			"NEW_KEY": "new_value",
		}))

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		content := string(data)
		assert.NotContains(t, content, "OLD_KEY")
		assert.Contains(t, content, "NEW_KEY=new_value")
	})

	t.Run("round-trips through ReadEnv", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")

		m := &FileEnvManager{Path: path}
		original := map[string]string{
			"PORT":     "3000",
			"HOST":     "localhost",
			"NODE_ENV": "production",
		}
		require.NoError(t, m.WriteEnvFile(original))

		readBack, err := m.ReadEnv()
		require.NoError(t, err)
		assert.Equal(t, original, readBack)
	})

	t.Run("empty map writes empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")

		m := &FileEnvManager{Path: path}
		require.NoError(t, m.WriteEnvFile(map[string]string{}))

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "", string(data))
	})

	t.Run("sets file permissions to 0640", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.env")

		m := &FileEnvManager{Path: path}
		require.NoError(t, m.WriteEnvFile(map[string]string{"KEY": "val"}))

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0640), info.Mode().Perm())
	})
}
