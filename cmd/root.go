// Package cmd holds the Cobra command tree.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"go-imapsync/internal/config"
	"go-imapsync/internal/secret"
	"go-imapsync/internal/sync"
)

// Exit codes: 0 success, 1 runtime failure, 2 usage/config.
const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2
)

var errRuntime = errors.New("runtime failure")

// version is injected at build time via -ldflags.
var version = "dev"

var (
	flagHost1, flagUser1, flagPassword1 string
	flagHost2, flagUser2, flagPassword2 string
	flagPort1, flagPort2                int
	flagSSL1, flagSSL2                  bool
	flagNoSSL1, flagNoSSL2              bool
	flagTLS1, flagTLS2                  bool
	flagNoTLS1, flagNoTLS2              bool
	flagDry, flagJustFolders            bool
	flagSkipEmpty                       bool
	flagUseHeader                       []string
	flagLogFile                         string
	flagVerbose                         bool
	flagInsecureTLS                     bool
	flagTimeout                         time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "go-imapsync",
	Short: "One-way IMAP mailbox sync (host1 → host2), imapsync-style",
	Long: `go-imapsync copies folders and messages from one IMAP account to another
without duplicates (Message-Id + Received by default).

Behavior reference: classic Perl imapsync.
Go patterns adapted from https://github.com/jniltinho/go-getmail.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runSync,
}

func init() {
	// Defaults: SSL on (IMAPS 993), matching imapsync “ssl if possible”.
	flagSSL1, flagSSL2 = true, true

	f := rootCmd.Flags()
	f.StringVar(&flagHost1, "host1", "", "source IMAP server")
	f.StringVar(&flagUser1, "user1", "", "source username")
	f.StringVar(&flagPassword1, "password1", "", "source password (or GOIMAPSYNC_PASSWORD1)")
	f.IntVar(&flagPort1, "port1", 0, "source port (default 993 if SSL, 143 otherwise)")
	f.BoolVar(&flagSSL1, "ssl1", true, "use SSL/IMAPS on host1")
	f.BoolVar(&flagNoSSL1, "nossl1", false, "disable SSL on host1")
	f.BoolVar(&flagTLS1, "tls1", false, "use STARTTLS on host1 (when not SSL)")
	f.BoolVar(&flagNoTLS1, "notls1", false, "do not use STARTTLS on host1")

	f.StringVar(&flagHost2, "host2", "", "destination IMAP server")
	f.StringVar(&flagUser2, "user2", "", "destination username")
	f.StringVar(&flagPassword2, "password2", "", "destination password (or GOIMAPSYNC_PASSWORD2)")
	f.IntVar(&flagPort2, "port2", 0, "destination port")
	f.BoolVar(&flagSSL2, "ssl2", true, "use SSL/IMAPS on host2")
	f.BoolVar(&flagNoSSL2, "nossl2", false, "disable SSL on host2")
	f.BoolVar(&flagTLS2, "tls2", false, "use STARTTLS on host2")
	f.BoolVar(&flagNoTLS2, "notls2", false, "do not use STARTTLS on host2")

	f.BoolVar(&flagDry, "dry", false, "do not change host2 (no CREATE/APPEND)")
	f.BoolVar(&flagJustFolders, "justfolders", false, "only create folders, skip messages")
	f.BoolVar(&flagSkipEmpty, "skipemptyfolders", false, "do not create empty host1 folders on host2")
	f.StringArrayVar(&flagUseHeader, "useheader", nil, "header(s) for message identity (default Message-Id, Received)")
	f.StringVar(&flagLogFile, "logfile", "", "also write log to this file")
	f.BoolVar(&flagVerbose, "verbose", false, "debug logging")
	f.BoolVar(&flagInsecureTLS, "insecuretls", false, "skip TLS certificate verification (lab only)")
	f.DurationVar(&flagTimeout, "timeout", 60*time.Second, "network timeout")

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("go-imapsync {{.Version}}\n")
}

// Execute runs the CLI and exits the process with the proper exit code.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		os.Exit(exitOK)
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	if errors.Is(err, errRuntime) {
		os.Exit(exitRuntime)
	}
	os.Exit(exitUsage)
}

func buildConfig() (*config.Config, error) {
	ssl1 := flagSSL1 && !flagNoSSL1
	ssl2 := flagSSL2 && !flagNoSSL2
	tls1 := !ssl1 && flagTLS1 && !flagNoTLS1
	tls2 := !ssl2 && flagTLS2 && !flagNoTLS2
	// If SSL off and TLS not explicitly disabled, prefer STARTTLS on 143.
	if !ssl1 && !flagNoTLS1 && !flagTLS1 {
		tls1 = true
	}
	if !ssl2 && !flagNoTLS2 && !flagTLS2 {
		tls2 = true
	}

	cfg := &config.Config{
		Host1: config.Side{
			Host:     flagHost1,
			Port:     flagPort1,
			User:     flagUser1,
			Password: secret.New(flagPassword1),
			SSL:      ssl1,
			TLS:      tls1,
		},
		Host2: config.Side{
			Host:     flagHost2,
			Port:     flagPort2,
			User:     flagUser2,
			Password: secret.New(flagPassword2),
			SSL:      ssl2,
			TLS:      tls2,
		},
		Dry:              flagDry,
		JustFolders:      flagJustFolders,
		SkipEmptyFolders: flagSkipEmpty,
		UseHeader:        flagUseHeader,
		LogFile:          flagLogFile,
		Verbose:          flagVerbose,
		Timeout:          flagTimeout,
		InsecureTLS:      flagInsecureTLS,
	}
	cfg.Defaults()
	cfg.ApplyEnvPasswords()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func newLogger(verbose bool, logFile string) (*slog.Logger, func(), error) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	var writers []io.Writer
	writers = append(writers, os.Stderr)
	cleanup := func() {}
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, nil, fmt.Errorf("logfile: %w", err)
		}
		writers = append(writers, f)
		cleanup = func() { _ = f.Close() }
	}
	w := io.MultiWriter(writers...)
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})), cleanup, nil
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := buildConfig()
	if err != nil {
		return err
	}
	log, cleanup, err := newLogger(cfg.Verbose, cfg.LogFile)
	if err != nil {
		return err
	}
	defer cleanup()

	log.Info("go-imapsync starting",
		"version", version,
		"host1", cfg.Host1.Host,
		"user1", cfg.Host1.User,
		"host2", cfg.Host2.Host,
		"user2", cfg.Host2.User,
		"dry", cfg.Dry,
		"justfolders", cfg.JustFolders,
	)

	runner := &sync.Runner{Cfg: cfg, Log: log}
	stats, err := runner.Run(cmd.Context())
	if stats != nil {
		stats.WriteSummary(os.Stderr)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", errRuntime, err)
	}
	if stats != nil && !stats.OK() {
		return fmt.Errorf("%w: %d message error(s)", errRuntime, stats.Failed)
	}
	return nil
}
