package cmd

import "github.com/spf13/viper"

// registerPathDefaults sets Viper defaults for all configurable paths.
// These are the lowest priority — overridden by config file, env vars, or flags.
func registerPathDefaults(v *viper.Viper) {
	v.SetDefault("app-dir", AppDir)
	v.SetDefault("data-dir", DataDir)
	v.SetDefault("libexec-dir", LibExecDir)
	v.SetDefault("config-dir", ConfigDir)
	v.SetDefault("cert-dir", CertDir)
	v.SetDefault("log-dir", LogDir)
	v.SetDefault("env-file", EnvFile)
}

// NewPathsFromViper resolves all paths from a Viper instance.
// Uses the priority chain: CLI flag > env var > config file > compile-time default.
func NewPathsFromViper(v *viper.Viper) Paths {
	return Paths{
		AppDir:     v.GetString("app-dir"),
		DataDir:    v.GetString("data-dir"),
		LibExecDir: v.GetString("libexec-dir"),
		ConfigDir:  v.GetString("config-dir"),
		CertDir:    v.GetString("cert-dir"),
		LogDir:     v.GetString("log-dir"),
		EnvFile:    v.GetString("env-file"),
	}
}
