// Package imaperr classifies IMAP/network errors into operator-friendly kinds.
//
// Use [Classify] before logging or aborting a sync loop. Fatal kinds
// (quota, closed connection, auth, TLS) should stop further APPEND attempts
// and surface a clear re-run hint in the end-of-run summary.
package imaperr

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

// Kind is a high-level error category for logs and summaries.
type Kind int

const (
	// KindUnknown is an unclassified error.
	KindUnknown Kind = iota
	// KindQuota is an IMAP OVERQUOTA / mailbox full rejection.
	KindQuota
	// KindClosed is a dead TCP/IMAP session (including EOF and reset).
	KindClosed
	// KindAuth is a failed LOGIN/AUTHENTICATE.
	KindAuth
	// KindTimeout is a network or context deadline.
	KindTimeout
	// KindTLS is a certificate or TLS handshake failure.
	KindTLS
	// KindNotFound is a missing mailbox or message.
	KindNotFound
	// KindServerNO is a generic IMAP NO from the server.
	KindServerNO
	// KindCanceled is a canceled context or interrupt.
	KindCanceled
)

// String returns a stable machine-readable kind name for summaries.
func (k Kind) String() string {
	switch k {
	case KindQuota:
		return "quota"
	case KindClosed:
		return "connection_closed"
	case KindAuth:
		return "authentication"
	case KindTimeout:
		return "timeout"
	case KindTLS:
		return "tls"
	case KindNotFound:
		return "not_found"
	case KindServerNO:
		return "server_rejected"
	case KindCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// Info is a classified error with a short operator-facing message.
type Info struct {
	Kind    Kind
	Message string // short human text (no password material)
	Hint    string // what to do next
	Fatal   bool   // abort folder / discourage continuing blindly
}

// Classify maps an error to Kind plus friendly Message/Hint text.
// It never expects password material in err; callers must not wrap secrets.
func Classify(err error) Info {
	if err == nil {
		return Info{}
	}
	if errors.Is(err, contextCanceled) || errors.Is(err, contextDeadline) {
		// fallback string match below if not standard context
	}
	msg := err.Error()
	low := strings.ToLower(msg)

	switch {
	case strings.Contains(low, "overquota") ||
		strings.Contains(low, "quota exceeded") ||
		strings.Contains(low, "mailbox is full") ||
		strings.Contains(low, "mailbox for user is full"):
		return Info{
			Kind:    KindQuota,
			Message: "destination mailbox quota exceeded (server refused APPEND)",
			Hint:    "free space or raise the host2 mailbox quota, then re-run (already copied messages are skipped)",
			Fatal:   true,
		}
	case strings.Contains(low, "use of closed network connection") ||
		strings.Contains(low, "connection reset") ||
		strings.Contains(low, "broken pipe") ||
		strings.Contains(low, "eof") && (strings.Contains(low, "read") || strings.Contains(low, "write")) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.EOF):
		return Info{
			Kind:    KindClosed,
			Message: "IMAP connection closed during the operation",
			Hint:    "often follows quota/errors or idle timeout; re-run to resume; reconnect is not automatic in this version",
			Fatal:   true,
		}
	case strings.Contains(low, "authentication failed") ||
		strings.Contains(low, "auth") && strings.Contains(low, "fail") ||
		strings.Contains(low, "invalid credentials") ||
		strings.Contains(low, "login failed"):
		return Info{
			Kind:    KindAuth,
			Message: "authentication failed",
			Hint:    "check username/password (or app password) for that host; passwords are never logged",
			Fatal:   true,
		}
	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline exceeded") ||
		strings.Contains(low, "i/o timeout"):
		return Info{
			Kind:    KindTimeout,
			Message: "network or IMAP timeout",
			Hint:    "increase --timeout, check network stability, then re-run",
			Fatal:   false,
		}
	case strings.Contains(low, "tls") || strings.Contains(low, "certificate") ||
		strings.Contains(low, "x509"):
		return Info{
			Kind:    KindTLS,
			Message: "TLS/SSL error while connecting or talking to the server",
			Hint:    "verify host certificate, port (993 IMAPS), and --ssl/--tls flags; --insecuretls is lab-only",
			Fatal:   true,
		}
	case strings.Contains(low, "not found") || strings.Contains(low, "no such"):
		return Info{
			Kind:    KindNotFound,
			Message: "message or mailbox not found",
			Hint:    "folder may have changed during sync; re-run usually safe",
			Fatal:   false,
		}
	case strings.Contains(low, "canceled") || strings.Contains(low, "cancelled"):
		return Info{
			Kind:    KindCanceled,
			Message: "operation canceled",
			Hint:    "interrupted by signal or parent context",
			Fatal:   true,
		}
	case strings.Contains(low, "imap: no") || strings.Contains(low, " no ["):
		return Info{
			Kind:    KindServerNO,
			Message: "IMAP server rejected the command",
			Hint:    "see detail in logs; common causes: ACL, message too large, policy",
			Fatal:   false,
		}
	default:
		return Info{
			Kind:    KindUnknown,
			Message: "unexpected IMAP/network error",
			Hint:    "see detail; re-run after fixing the underlying issue (duplicates are skipped)",
			Fatal:   false,
		}
	}
}

// sentinel placeholders resolved via string match mostly; keep errors.Is friendly for context
var (
	contextCanceled = errString("context canceled")
	contextDeadline = errString("context deadline exceeded")
)

type errString string

func (e errString) Error() string { return string(e) }

// Format builds a one-line operator message including side and operation
// (e.g. "host2 APPEND: destination mailbox quota exceeded").
func Format(side, op string, err error) string {
	if err == nil {
		return ""
	}
	info := Classify(err)
	return fmt.Sprintf("%s %s: %s", side, op, info.Message)
}

// Detail returns the original error text trimmed for logs (secondary field).
// Timing suffixes from some IMAP libraries are stripped for readability.
func Detail(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	// collapse noisy imap timing suffix if present
	if i := strings.Index(s, " (0."); i > 0 {
		s = s[:i]
	}
	return s
}
