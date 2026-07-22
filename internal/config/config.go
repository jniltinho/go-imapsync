// Package config holds validated runtime options derived from CLI flags.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go-imapsync/internal/secret"
)

// Side is one IMAP endpoint (source host1 or destination host2).
type Side struct {
	Host     string
	Port     int
	User     string
	Password secret.String
	// SSL forces IMAPS (implicit TLS). When false and TLS is true, STARTTLS is used.
	SSL bool
	// TLS enables STARTTLS when SSL is false. Ignored when SSL is true.
	TLS bool
}

// Config is the full sync configuration.
type Config struct {
	Host1 Side
	Host2 Side

	Dry              bool
	JustFolders      bool
	SkipEmptyFolders bool
	UseHeader        []string
	LogFile          string
	Verbose          bool
	Timeout          time.Duration
	InsecureTLS      bool // skip cert verify (lab only)
}

// Defaults fills ports and crypto defaults when unset.
func (c *Config) Defaults() {
	if len(c.UseHeader) == 0 {
		c.UseHeader = []string{"Message-Id", "Received"}
	}
	if c.Timeout <= 0 {
		c.Timeout = 60 * time.Second
	}
	applySideDefaults(&c.Host1)
	applySideDefaults(&c.Host2)
}

func applySideDefaults(s *Side) {
	// Prefer SSL (IMAPS) by default when neither SSL nor TLS was forced off.
	// Callers set SSL=true as the CLI default.
	if s.Port == 0 {
		if s.SSL {
			s.Port = 993
		} else {
			s.Port = 143
		}
	}
}

// ApplyEnvPasswords fills empty passwords from environment variables.
func (c *Config) ApplyEnvPasswords() {
	if c.Host1.Password.IsZero() {
		if v := os.Getenv("GOIMAPSYNC_PASSWORD1"); v != "" {
			c.Host1.Password = secret.New(v)
		}
	}
	if c.Host2.Password.IsZero() {
		if v := os.Getenv("GOIMAPSYNC_PASSWORD2"); v != "" {
			c.Host2.Password = secret.New(v)
		}
	}
}

// Validate checks required fields after Defaults and ApplyEnvPasswords.
func (c *Config) Validate() error {
	var missing []string
	if c.Host1.Host == "" {
		missing = append(missing, "--host1")
	}
	if c.Host1.User == "" {
		missing = append(missing, "--user1")
	}
	if c.Host1.Password.IsZero() {
		missing = append(missing, "--password1 or GOIMAPSYNC_PASSWORD1")
	}
	if c.Host2.Host == "" {
		missing = append(missing, "--host2")
	}
	if c.Host2.User == "" {
		missing = append(missing, "--user2")
	}
	if c.Host2.Password.IsZero() {
		missing = append(missing, "--password2 or GOIMAPSYNC_PASSWORD2")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required options: %s", strings.Join(missing, ", "))
	}
	return nil
}
