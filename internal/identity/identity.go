// Package identity builds imapsync-style message keys from headers.
//
// Keys are used to decide whether a host1 message already exists on host2.
// Unlike go-getmail's UIDVALIDITY:UID oldmail model, identity is derived from
// message headers so re-runs remain safe across servers. Default fields are
// Message-Id and Received (not IMAP UID). Size alone is never used as identity.
package identity

import (
	"bufio"
	"bytes"
	"net/textproto"
	"strings"
)

// KeyFromHeaders builds a stable identity key from raw RFC822 headers
// using the given header field names (case-insensitive).
//
// Multiple values for the same field (e.g. several Received lines) are included
// in order. Whitespace is normalized. Empty fields are skipped.
// If no selected field has a value, KeyFromHeaders returns "".
//
// When fields is nil or empty, Message-Id and Received are used.
func KeyFromHeaders(rawHeaders []byte, fields []string) string {
	if len(fields) == 0 {
		fields = []string{"Message-Id", "Received"}
	}
	hdr, err := textproto.NewReader(bufio.NewReader(bytes.NewReader(rawHeaders))).ReadMIMEHeader()
	if err != nil && len(hdr) == 0 {
		// Fall back to line-oriented parse for truncated fixtures.
		hdr = parseLoose(rawHeaders)
	}
	var parts []string
	for _, f := range fields {
		values := hdr.Values(f)
		if len(values) == 0 {
			// Canonical MIME maps Message-Id -> Message-Id, but also try Message-ID.
			values = hdr.Values(textproto.CanonicalMIMEHeaderKey(f))
		}
		for _, v := range values {
			v = normalize(v)
			if v != "" {
				parts = append(parts, strings.ToLower(f)+":"+v)
			}
		}
	}
	return strings.Join(parts, "|")
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// parseLoose is a best-effort header parser when MIME reader fails.
func parseLoose(raw []byte) textproto.MIMEHeader {
	h := make(textproto.MIMEHeader)
	sc := bufio.NewScanner(bytes.NewReader(raw))
	var key string
	var val strings.Builder
	flush := func() {
		if key == "" {
			return
		}
		h.Add(key, val.String())
		key = ""
		val.Reset()
	}
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			break
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if key != "" {
				val.WriteByte(' ')
				val.WriteString(strings.TrimSpace(line))
			}
			continue
		}
		flush()
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			continue
		}
		key = textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(line[:i]))
		val.WriteString(strings.TrimSpace(line[i+1:]))
	}
	flush()
	return h
}

// SplitHeadersBody splits a raw RFC822 message into headers and body at the
// first blank line (CRLF or LF). If no separator is found, the entire message
// is returned as headers and body is nil.
func SplitHeadersBody(msg []byte) (headers, body []byte) {
	// Prefer CRLF then LF.
	if i := bytes.Index(msg, []byte("\r\n\r\n")); i >= 0 {
		return msg[:i+4], msg[i+4:]
	}
	if i := bytes.Index(msg, []byte("\n\n")); i >= 0 {
		return msg[:i+2], msg[i+2:]
	}
	return msg, nil
}
