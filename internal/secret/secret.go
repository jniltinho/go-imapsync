// Package secret holds credential material and obtains it from external
// commands, guaranteeing that secret values never leak into logs, error
// messages, or serialized output.
//
// Adapted from github.com/jniltinho/go-getmail internal/secret.
//
// Use [New] for passwords from flags or env, [FromCommand] for password_command
// style retrieval, and always pass [String] to logging as attributes (LogValue
// redacts). Call [String.Reveal] only at the AUTH/LOGIN call site.
package secret

import "log/slog"

const redacted = "[redacted]"

// String is a password value whose every printable representation is
// redacted. Use Reveal to obtain the real value at the point of use
// (e.g. the AUTH command of a protocol client).
type String struct {
	value string
}

// New wraps a raw secret value.
func New(value string) String { return String{value: value} }

// Reveal returns the underlying secret value. Prefer keeping the lifetime of
// the returned string as short as possible.
func (s String) Reveal() string { return s.value }

// IsZero reports whether no secret is set.
func (s String) IsZero() bool { return s.value == "" }

// String implements fmt.Stringer and always returns a redacted token.
func (s String) String() string { return redacted }

// GoString implements fmt.GoStringer and always returns a redacted token.
func (s String) GoString() string { return redacted }

// LogValue implements slog.LogValuer so structured logs never print the secret.
func (s String) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalText implements encoding.TextMarshaler with a redacted value.
func (s String) MarshalText() ([]byte, error) { return []byte(redacted), nil }

// UnmarshalText implements encoding.TextUnmarshaler so configuration formats
// can populate a String from a password field.
func (s *String) UnmarshalText(text []byte) error {
	s.value = string(text)
	return nil
}
