// Package imapclient wraps emersion/go-imap/v2 for dual-host sync.
// Dial/login patterns adapted from github.com/jniltinho/go-getmail.
package imapclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"go-imapsync/internal/config"
	"go-imapsync/internal/secret"
)

// Client is one authenticated IMAP session.
type Client struct {
	label    string
	side     config.Side
	timeout  time.Duration
	insecure bool
	tlsCfg   *tls.Config

	c           *imapclient.Client
	delimiter   rune
	numMessages uint32
	selected    string
}

// Options configures a new client.
type Options struct {
	Label     string
	Side      config.Side
	Timeout   time.Duration
	Insecure  bool
	TLSConfig *tls.Config // tests
}

// Dial connects and authenticates.
func Dial(ctx context.Context, opt Options) (*Client, error) {
	if opt.Timeout <= 0 {
		opt.Timeout = 60 * time.Second
	}
	cl := &Client{
		label:    opt.Label,
		side:     opt.Side,
		timeout:  opt.Timeout,
		insecure: opt.Insecure,
		tlsCfg:   opt.TLSConfig,
	}
	if err := cl.connect(ctx); err != nil {
		return nil, err
	}
	return cl, nil
}

func (c *Client) connect(ctx context.Context) error {
	addr := net.JoinHostPort(c.side.Host, strconv.Itoa(c.side.Port))
	tlsConfig := c.tlsCfg
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			ServerName:         c.side.Host,
			InsecureSkipVerify: c.insecure, //nolint:gosec // explicit lab flag only
		}
	}

	dialer := &net.Dialer{Timeout: c.timeout}
	opts := &imapclient.Options{TLSConfig: tlsConfig, Dialer: dialer}

	var (
		client *imapclient.Client
		err    error
	)

	switch {
	case c.side.SSL:
		conn, dErr := (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", addr)
		if dErr != nil {
			return fmt.Errorf("%s dial TLS %s: %w", c.label, addr, dErr)
		}
		client = imapclient.New(conn, opts)
	case c.side.TLS:
		conn, dErr := dialer.DialContext(ctx, "tcp", addr)
		if dErr != nil {
			return fmt.Errorf("%s dial %s: %w", c.label, addr, dErr)
		}
		client, err = imapclient.NewStartTLS(conn, opts)
		if err != nil {
			return fmt.Errorf("%s STARTTLS %s: %w", c.label, addr, err)
		}
	default:
		conn, dErr := dialer.DialContext(ctx, "tcp", addr)
		if dErr != nil {
			return fmt.Errorf("%s dial plain %s: %w", c.label, addr, dErr)
		}
		client = imapclient.New(conn, opts)
	}

	if err := client.Login(c.side.User, c.side.Password.Reveal()).Wait(); err != nil {
		client.Close()
		// Do not include password in error.
		return fmt.Errorf("%s authentication failed for user %q: %w", c.label, c.side.User, err)
	}
	c.c = client
	return nil
}

// Close logs out and closes the connection.
func (c *Client) Close() error {
	if c == nil || c.c == nil {
		return nil
	}
	defer func() {
		c.c.Close()
		c.c = nil
	}()
	if err := c.c.Logout().Wait(); err != nil {
		return fmt.Errorf("%s logout: %w", c.label, err)
	}
	return nil
}

// Password returns the redacted secret (for logging only via LogValue).
func (c *Client) Password() secret.String { return c.side.Password }

// Folder is a mailbox from LIST.
type Folder struct {
	Name       string
	Delimiter  rune
	NoSelect   bool
	Attributes []imap.MailboxAttr
}

// ListFolders lists all mailboxes under the root.
func (c *Client) ListFolders(ctx context.Context) ([]Folder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mailboxes, err := c.c.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("%s LIST: %w", c.label, err)
	}
	out := make([]Folder, 0, len(mailboxes))
	for _, m := range mailboxes {
		if m.Delim != 0 {
			c.delimiter = m.Delim
		}
		f := Folder{
			Name:       m.Mailbox,
			Delimiter:  m.Delim,
			Attributes: m.Attrs,
		}
		for _, a := range m.Attrs {
			if a == imap.MailboxAttrNoSelect {
				f.NoSelect = true
				break
			}
		}
		out = append(out, f)
	}
	return out, nil
}

// Delimiter returns the last seen hierarchy delimiter (or '/').
func (c *Client) Delimiter() rune {
	if c.delimiter == 0 {
		return '/'
	}
	return c.delimiter
}

// CreateFolder creates a mailbox on this server.
func (c *Client) CreateFolder(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.c.Create(name, nil).Wait(); err != nil {
		return fmt.Errorf("%s CREATE %q: %w", c.label, name, err)
	}
	return nil
}

// Select opens a mailbox for read-write (or read-only if readOnly).
func (c *Client) Select(ctx context.Context, name string, readOnly bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	opts := &imap.SelectOptions{ReadOnly: readOnly}
	data, err := c.c.Select(name, opts).Wait()
	if err != nil {
		return fmt.Errorf("%s SELECT %q: %w", c.label, name, err)
	}
	c.selected = name
	c.numMessages = data.NumMessages
	return nil
}

// MessageMeta is identity material plus body for one message.
type MessageMeta struct {
	UID          imap.UID
	Flags        []imap.Flag
	InternalDate time.Time
	Headers      []byte
	Body         []byte // full RFC822 when fetched
	Size         int64
}

// FetchAllForIdentity loads UID, flags, date, and header fields for every
// message in the selected mailbox (for building identity sets).
func (c *Client) FetchAllForIdentity(ctx context.Context, headerFields []string) ([]MessageMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.numMessages == 0 {
		return nil, nil
	}
	var seqSet imap.SeqSet
	seqSet.AddRange(1, 0) // 1:*
	section := &imap.FetchItemBodySection{
		Specifier:    imap.PartSpecifierHeader,
		HeaderFields: headerFields,
		Peek:         true,
	}
	opts := &imap.FetchOptions{
		UID:          true,
		Flags:        true,
		InternalDate: true,
		RFC822Size:   true,
		BodySection:  []*imap.FetchItemBodySection{section},
	}
	msgs, err := c.c.Fetch(seqSet, opts).Collect()
	if err != nil {
		return nil, fmt.Errorf("%s FETCH headers: %w", c.label, err)
	}
	out := make([]MessageMeta, 0, len(msgs))
	for _, msg := range msgs {
		m := MessageMeta{
			UID:          msg.UID,
			Flags:        msg.Flags,
			InternalDate: msg.InternalDate,
			Size:         msg.RFC822Size,
		}
		for _, bs := range msg.BodySection {
			if len(bs.Bytes) > 0 {
				m.Headers = bs.Bytes
				break
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// FetchFull retrieves full RFC822 for one UID in the selected mailbox.
func (c *Client) FetchFull(ctx context.Context, uid imap.UID) (*MessageMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	opts := &imap.FetchOptions{
		UID:          true,
		Flags:        true,
		InternalDate: true,
		RFC822Size:   true,
		BodySection:  []*imap.FetchItemBodySection{{}},
	}
	cmd := c.c.Fetch(imap.UIDSetNum(uid), opts)
	msg := cmd.Next()
	if msg == nil {
		cmd.Close()
		return nil, fmt.Errorf("%s FETCH UID %d: not found", c.label, uid)
	}
	m := &MessageMeta{}
	for {
		item := msg.Next()
		if item == nil {
			break
		}
		switch data := item.(type) {
		case imapclient.FetchItemDataUID:
			m.UID = data.UID
		case imapclient.FetchItemDataFlags:
			m.Flags = data.Flags
		case imapclient.FetchItemDataInternalDate:
			m.InternalDate = data.Time
		case imapclient.FetchItemDataRFC822Size:
			m.Size = data.Size
		case imapclient.FetchItemDataBodySection:
			b, err := io.ReadAll(data.Literal)
			if err != nil {
				cmd.Close()
				return nil, err
			}
			m.Body = b
		}
	}
	if err := cmd.Close(); err != nil {
		return nil, fmt.Errorf("%s FETCH close: %w", c.label, err)
	}
	return m, nil
}

// Append uploads a message to mailbox with flags and internal date.
func (c *Client) Append(ctx context.Context, mailbox string, body []byte, flags []imap.Flag, date time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	opts := &imap.AppendOptions{Flags: flags, Time: date}
	cmd := c.c.Append(mailbox, int64(len(body)), opts)
	if _, err := cmd.Write(body); err != nil {
		cmd.Close()
		return fmt.Errorf("%s APPEND write %q: %w", c.label, mailbox, err)
	}
	if err := cmd.Close(); err != nil {
		return fmt.Errorf("%s APPEND close %q: %w", c.label, mailbox, err)
	}
	if _, err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s APPEND wait %q: %w", c.label, mailbox, err)
	}
	return nil
}

// FolderExists reports whether name is present in LIST results (caller may pass cache).
func FolderExists(folders []Folder, name string) bool {
	for _, f := range folders {
		if f.Name == name {
			return true
		}
	}
	return false
}
