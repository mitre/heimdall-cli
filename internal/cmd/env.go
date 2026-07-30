package cmd

import (
	"os"
	"sort"
	"strings"
)

// FileEnvManager implements EnvManager backed by a real file.
type FileEnvManager struct {
	Path string
}

// NewFileEnvManager returns an EnvManager backed by the compile-time default
// env file path. Runners that need a custom path should set it via Paths.EnvFile.
func NewFileEnvManager() *FileEnvManager {
	return &FileEnvManager{Path: EnvFile}
}

func (m *FileEnvManager) GetEnvFilePath() string { return m.Path }

// ReadEnv parses KEY=VALUE and KEY="VALUE" lines, skipping comments and blanks.
// Always returns a non-nil map, even on error, so callers can safely access
// keys without nil checks. The error is still returned for callers that care.
func (m *FileEnvManager) ReadEnv() (map[string]string, error) {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return map[string]string{}, err
	}
	return ParseEnv(string(data)), nil
}

// ParseEnv parses env file content into a map.
func ParseEnv(content string) map[string]string {
	env := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, "\"'")
		env[key] = val
	}
	return env
}

// WriteEnvKey updates or appends a key in the env file, preserving comments and order.
func (m *FileEnvManager) WriteEnvKey(key, value string) error {
	var lines []string
	data, err := os.ReadFile(m.Path)
	if err == nil {
		lines = strings.Split(string(data), "\n")
	}

	found := false
	var out []string
	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped != "" && !strings.HasPrefix(stripped, "#") && strings.Contains(stripped, "=") {
			k := strings.TrimSpace(stripped[:strings.Index(stripped, "=")])
			if k == key {
				out = append(out, key+"="+value)
				found = true
				continue
			}
		}
		out = append(out, line)
	}
	if !found {
		out = append(out, key+"="+value)
	}

	// Ensure file ends with newline
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(m.Path, []byte(content), 0640)
}

// WriteEnvFile writes all entries to the env file (replacing contents).
// Keys are written in sorted order for deterministic output.
func (m *FileEnvManager) WriteEnvFile(entries map[string]string) error {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k + "=" + entries[k] + "\n")
	}
	return os.WriteFile(m.Path, []byte(sb.String()), 0640)
}

// MaskSecret masks a secret value for display: first 3 + *** + last 3.
func MaskSecret(val string) string {
	if val == "" {
		return "(empty)"
	}
	if len(val) <= 6 {
		return "***"
	}
	return val[:3] + "***" + val[len(val)-3:]
}
