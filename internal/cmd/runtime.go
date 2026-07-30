package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// execRunner implements ExecRunner using os/exec.
type execRunner struct{}

func (r *execRunner) Run(name string, args ...string) (string, int, error) {
	return r.RunWithEnv(nil, name, args...)
}

func (r *execRunner) RunWithEnv(env map[string]string, name string, args ...string) (string, int, error) {
	c := exec.Command(name, args...)
	if env != nil {
		c.Env = os.Environ()
		for k, v := range env {
			c.Env = append(c.Env, k+"="+v)
		}
	}
	out, err := c.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode(), nil
		}
		return string(out), 1, err
	}
	return string(out), 0, nil
}

func (r *execRunner) RunWithStdin(stdin, name string, args ...string) (string, int, error) {
	c := exec.Command(name, args...)
	c.Stdin = strings.NewReader(stdin)
	out, err := c.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode(), nil
		}
		return string(out), 1, err
	}
	return string(out), 0, nil
}

// systemdRunner implements SystemdRunner using systemctl.
type systemdRunner struct{}

func (r *systemdRunner) IsActive(service string) (bool, error) {
	c := exec.Command("systemctl", "is-active", service)
	out, err := c.Output()
	return strings.TrimSpace(string(out)) == "active", err
}
func (r *systemdRunner) Start(service string) error {
	return exec.Command("systemctl", "start", service).Run()
}
func (r *systemdRunner) Stop(service string) error {
	return exec.Command("systemctl", "stop", service).Run()
}
func (r *systemdRunner) Restart(service string) error {
	return exec.Command("systemctl", "restart", service).Run()
}
func (r *systemdRunner) Enable(service string) error {
	return exec.Command("systemctl", "enable", service).Run()
}
func (r *systemdRunner) EnableNow(service string) error {
	return exec.Command("systemctl", "enable", "--now", service).Run()
}
func (r *systemdRunner) Reload(service string) error {
	return exec.Command("systemctl", "reload", service).Run()
}
func (r *systemdRunner) ShowProperty(service, property string) (string, error) {
	c := exec.Command("systemctl", "show", "-p", property, "--value", service)
	out, err := c.Output()
	return strings.TrimSpace(string(out)), err
}

// osFileSystem implements FileSystem using real OS operations.
type osFileSystem struct{}

func (r *osFileSystem) ReadFile(path string) ([]byte, error)  { return os.ReadFile(path) }
func (r *osFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (r *osFileSystem) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }
func (r *osFileSystem) CopyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode())
}
func (r *osFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
func (r *osFileSystem) TempDir(prefix string) (string, error) {
	return os.MkdirTemp("", prefix)
}
func (r *osFileSystem) RemoveAll(path string) error { return os.RemoveAll(path) }
func (r *osFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *osFileSystem) FindFiles(root, namePattern string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Tolerate per-entry errors (permissions, transient) — skip them.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ok, matchErr := filepath.Match(namePattern, d.Name())
		if matchErr != nil {
			return matchErr
		}
		if ok {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		// Root missing is not an error for our use case — return empty.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return matches, nil
}

// psqlConnector implements DBConnector using psql.
type psqlConnector struct{}

func (r *psqlConnector) TestConnection(host string, port int, user, password, dbName string) error {
	env := map[string]string{"PGPASSWORD": password}
	c := exec.Command("psql", "-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", dbName, "-c", "SELECT 1")
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, k+"="+v)
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("connection failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *psqlConnector) TableCount(host string, port int, user, password, dbName string) (int, error) {
	env := map[string]string{"PGPASSWORD": password}
	c := exec.Command("psql", "-h", host, "-p", strconv.Itoa(port), "-U", user, "-d", dbName,
		"-t", "-A", "-c", "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, k+"="+v)
	}
	out, err := c.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("query failed: %s", strings.TrimSpace(string(out)))
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("unexpected output: %s", string(out))
	}
	return count, nil
}

// initIO sets Out and ErrOut from the Cobra command if they are nil.
// This eliminates the repeated nil-check boilerplate in every RunE closure.
func initIO(out, errOut *io.Writer, cmd *cobra.Command) {
	if *out == nil {
		*out = cmd.OutOrStdout()
	}
	if errOut != nil && *errOut == nil {
		*errOut = cmd.ErrOrStderr()
	}
}

// envDefault returns the value of key from env map, or fallback if not present/empty.
func envDefault(env map[string]string, key, fallback string) string {
	if v, ok := env[key]; ok && v != "" {
		return v
	}
	return fallback
}

// huhPrompter implements Prompter using charm.land/huh/v2.
// Falls back to accessible mode (plain text stdin/stdout) when not a TTY.
type huhPrompter struct{}

func (p *huhPrompter) newForm(groups ...*huh.Group) *huh.Form {
	f := huh.NewForm(groups...)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		f = f.WithAccessible(true)
	}
	return f
}

func (p *huhPrompter) Input(prompt string, defaultValue string) (string, error) {
	result := defaultValue
	err := p.newForm(
		huh.NewGroup(
			huh.NewInput().Title(prompt).Value(&result),
		),
	).Run()
	return result, err
}

func (p *huhPrompter) Password(prompt string) (string, error) {
	var result string
	err := p.newForm(
		huh.NewGroup(
			huh.NewInput().Title(prompt).EchoMode(huh.EchoModePassword).Value(&result),
		),
	).Run()
	return result, err
}

func (p *huhPrompter) Confirm(prompt string, defaultValue bool) (bool, error) {
	result := defaultValue
	err := p.newForm(
		huh.NewGroup(
			huh.NewConfirm().Title(prompt).Value(&result),
		),
	).Run()
	return result, err
}

func (p *huhPrompter) Select(prompt string, defaultValue string, options []string) (int, error) {
	opts := make([]huh.Option[int], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, i)
	}
	var result int
	err := p.newForm(
		huh.NewGroup(
			huh.NewSelect[int]().Title(prompt).Options(opts...).Value(&result),
		),
	).Run()
	return result, err
}

func (p *huhPrompter) CanPrompt() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// stdioDetector checks if stdin is a real terminal.
type stdioDetector struct{}

func (r *stdioDetector) IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
