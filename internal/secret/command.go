package secret

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// FromCommand runs argv directly (no shell) and returns its stdout, with a
// single trailing newline trimmed, as the secret.
//
// The returned error never contains command stdout — only the command name and
// exit status or context error — so failed password helpers cannot leak
// secrets into logs.
func FromCommand(ctx context.Context, argv []string) (String, error) {
	if len(argv) == 0 {
		return String{}, fmt.Errorf("password_command is empty")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return String{}, fmt.Errorf("password_command %q: %w", argv[0], ctx.Err())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return String{}, fmt.Errorf("password_command %q exited with status %d", argv[0], exitErr.ExitCode())
		}
		return String{}, fmt.Errorf("password_command %q: %w", argv[0], err)
	}

	out := stdout.String()
	out = strings.TrimSuffix(out, "\n")
	return New(out), nil
}
