package cmd

import "strconv"

// DBConfig holds database connection parameters extracted from the environment.
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// PortStr returns the port as a string, suitable for passing to CLI tools
// like pg_dump and psql.
func (c DBConfig) PortStr() string {
	return strconv.Itoa(c.Port)
}

// PgEnv returns the environment map for passing PGPASSWORD to psql/pg_dump.
func (c DBConfig) PgEnv() map[string]string {
	return map[string]string{"PGPASSWORD": c.Password}
}

// ExtractDBConfig reads database connection parameters from the environment map,
// applying defaults for missing values.
func ExtractDBConfig(env map[string]string) DBConfig {
	cfg := DBConfig{
		Host:     envDefault(env, "DATABASE_HOST", "localhost"),
		User:     envDefault(env, "DATABASE_USERNAME", "postgres"),
		Password: env["DATABASE_PASSWORD"],
		DBName:   envDefault(env, "DATABASE_NAME", "heimdall-server-production"),
	}
	cfg.Port = DefaultDBPort
	if p, err := strconv.Atoi(envDefault(env, "DATABASE_PORT", "5432")); err == nil {
		cfg.Port = p
	}
	return cfg
}
