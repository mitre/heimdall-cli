package cmd

// fapolicyd is RHEL/Fedora-specific. The runner stays cross-platform
// compileable so unit tests with fakes work on any host; on non-Linux
// the runtime checks (no fapolicyd-cli in PATH, no /usr/share/heimdall-server)
// make every operation a no-op. Cobra wiring may gate command visibility
// per platform later (see saf-packaging-alk).

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// fapolicydTrustFile is the name of the trust list under /etc/fapolicyd/trust.d/.
// fapolicyd-cli writes to this file when called with --trust-file.
const fapolicydTrustFile = "heimdall-server"

// fapolicydAppDir is where the RPM lays down the bundled Node.js binary
// and native .node addon files.
const fapolicydAppDir = "/usr/share/heimdall-server"

// fapolicydService is the systemd unit name we check before reloading.
const fapolicydService = "fapolicyd"

// FapolicydRunner replaces heimdall-fapolicyd-trust.sh. It registers the
// bundled Node.js binary and native .node addon files with the fapolicyd
// trust database (and removes them on uninstall). All operations are
// tolerant — fapolicyd may not be installed at all on a given host.
type FapolicydRunner struct {
	Exec    ExecRunner
	FS      FileSystem
	Systemd SystemdRunner
	Out     io.Writer
}

// Add registers the bundled binary and native addons with fapolicyd's
// trust database, then reloads fapolicyd if it's running.
func (r *FapolicydRunner) Add() error {
	nodeBin := fapolicydAppDir + "/node"
	if r.FS.Exists(nodeBin) {
		r.trust(nodeBin)
	}

	addons, err := r.FS.FindFiles(fapolicydAppDir, "*.node")
	if err != nil {
		return err
	}
	for _, addon := range addons {
		r.trust(addon)
	}

	r.reloadIfActive()
	return nil
}

// Remove deletes the heimdall-server trust file and reloads fapolicyd
// if it's running. Tolerant to a missing trust file (fapolicyd may
// never have been configured with one).
func (r *FapolicydRunner) Remove() error {
	trustPath := "/etc/fapolicyd/trust.d/" + fapolicydTrustFile
	// RemoveAll is no-op-on-missing — same as `rm -f`.
	if err := r.FS.RemoveAll(trustPath); err != nil {
		return err
	}
	r.reloadIfActive()
	return nil
}

// trust adds a single path to the heimdall-server trust list. Errors
// are intentionally swallowed — fapolicyd-cli may be absent, the trust
// file may already contain the entry, etc. Matches the bash `|| true`.
func (r *FapolicydRunner) trust(path string) {
	_, _, _ = r.Exec.Run("fapolicyd-cli",
		"--file", "add", path,
		"--trust-file", fapolicydTrustFile)
}

// reloadIfActive triggers fapolicyd-cli --update only when the service
// is currently active. No-op (and silently swallows errors) otherwise.
func (r *FapolicydRunner) reloadIfActive() {
	active, err := r.Systemd.IsActive(fapolicydService)
	if err != nil || !active {
		return
	}
	_, _, _ = r.Exec.Run("fapolicyd-cli", "--update")
}

// NewFapolicydCmd builds the `fapolicyd` parent command with `add` and
// `remove` subcommands. Designed to be invoked from RPM scriptlets
// (%post / %postun) — not a daily admin tool.
func NewFapolicydCmd(runner *FapolicydRunner) *cobra.Command {
	if runner == nil {
		runner = &FapolicydRunner{
			Exec:    &execRunner{},
			FS:      &osFileSystem{},
			Systemd: &systemdRunner{},
		}
	}

	cmd := &cobra.Command{
		Use:   "fapolicyd",
		Short: "Manage fapolicyd trust entries for the bundled Node.js binary",
		Long: `Register the bundled Node.js binary and native .node addon files with
the fapolicyd trust database. Invoked from RPM %post / %postun
scriptlets; usually you do not run this by hand.

The trust list is written to /etc/fapolicyd/trust.d/heimdall-server.
If the fapolicyd service is running, it is reloaded after changes so
the new trust entries take effect immediately.

This command is a safe no-op on systems without fapolicyd installed.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:          "add",
		Short:        "Trust the bundled binary and addons",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			runner.Out = c.OutOrStdout()
			if err := runner.Add(); err != nil {
				return err
			}
			fmt.Fprintln(runner.Out, "fapolicyd: trust entries added")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:          "remove",
		Short:        "Remove trust entries (called on RPM uninstall)",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			runner.Out = c.OutOrStdout()
			if err := runner.Remove(); err != nil {
				return err
			}
			fmt.Fprintln(runner.Out, "fapolicyd: trust entries removed")
			return nil
		},
	})

	return cmd
}
