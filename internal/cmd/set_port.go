package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
)

// SetPortRunner holds dependencies for the set-port command.
type SetPortRunner struct {
	Exec      ExecRunner
	Env       EnvManager
	Systemd   SystemdRunner
	Out       io.Writer
	ErrOut    io.Writer
	CheckRoot func() error
}

// Run executes the set-port logic.
func (r *SetPortRunner) Run(portStr string) error {
	if r.CheckRoot != nil {
		if err := r.CheckRoot(); err != nil {
			return err
		}
	}
	p, err := strconv.Atoi(portStr)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("port must be 1-65535")
	}

	env, err := r.Env.ReadEnv()
	if err != nil {
		return fmt.Errorf("failed to read env: %w", err)
	}
	oldPort := env["PORT"]
	if oldPort == "" {
		oldPort = DefaultAppPort
	}

	if err := r.Env.WriteEnvKey("PORT", portStr); err != nil {
		return fmt.Errorf("failed to update env: %w", err)
	}
	fmt.Fprintf(r.Out, "Config:    PORT=%s (was %s)\n", portStr, oldPort)

	// SELinux
	r.Exec.Run("semanage", "port", "-a", "-t", "heimdall_server_port_t", "-p", "tcp", portStr)
	r.Exec.Run("semanage", "port", "-m", "-t", "heimdall_server_port_t", "-p", "tcp", portStr)
	fmt.Fprintf(r.Out, "SELinux:   port %s registered\n", portStr)

	// firewalld
	r.Exec.Run("firewall-cmd", "--permanent", "--add-port="+portStr+"/tcp")
	r.Exec.Run("firewall-cmd", "--reload")
	fmt.Fprintf(r.Out, "firewalld: port %s opened\n", portStr)

	// Restart
	fmt.Fprintln(r.Out)
	fmt.Fprintln(r.Out, "Restarting service...")
	if err := r.Systemd.Restart(ServiceName); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}
	fmt.Fprintf(r.Out, "Done. Heimdall now listening on port %s.\n", portStr)

	return nil
}

// NewSetPortCmd creates the set-port command.
func NewSetPortCmd(runner *SetPortRunner) *cobra.Command {
	if runner == nil {
		runner = &SetPortRunner{
			Exec:      &execRunner{},
			Env:       NewFileEnvManager(),
			Systemd:   &systemdRunner{},
			CheckRoot: requireRoot,
		}
	}
	return &cobra.Command{
		Use:   "set-port <port>",
		Short: "Change the listen port (updates config, SELinux, firewalld)",
		Long: `Change the port that Heimdall Server listens on. This command updates
three system layers in a single operation:

  1. Config:    Sets PORT in backend.env to the new value
  2. SELinux:   Registers the new port as heimdall_server_port_t so the
               service is allowed to bind under enforcing mode
  3. firewalld: Opens the new port in the permanent firewall rules

After updating all layers, the service is automatically restarted.
The port must be between 1 and 65535. Requires root privileges.`,
		Example: `  # Change the listen port to 8443
  sudo heimdall-cli set-port 8443

  # Change to a non-standard port
  sudo heimdall-cli set-port 9000`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			initIO(&runner.Out, &runner.ErrOut, cmd)
			return runner.Run(args[0])
		},
	}
}
