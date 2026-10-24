package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"time"
)

const defaultTimeout = 30 * time.Second

// ErrInvalidEncryption is returned for an unknown encryption mode.
var ErrInvalidEncryption = errors.New("invalid SMTP encryption type")

// Client represents an SMTP client.
type Client struct {
	host       string
	port       int
	username   string
	password   string
	encryption EncryptionType
	timeout    time.Duration
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithTimeout sets the timeout applied to connecting, TLS handshakes, and SMTP operations.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// NewClient creates a new SMTP client.
func NewClient(host string, port int, username, password string, encryption EncryptionType, opts ...ClientOption) *Client {
	c := &Client{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		encryption: encryption,
		timeout:    defaultTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// Send sends one or more messages using the SMTP client.
func (c *Client) Send(messages ...*Message) error {
	if c == nil {
		return errors.New("mail client must not be nil")
	}
	if c.host == "" {
		return errors.New("SMTP host is required")
	}
	if c.port < 1 || c.port > 65535 {
		return fmt.Errorf("invalid SMTP port %d", c.port)
	}
	if c.timeout <= 0 {
		return errors.New("SMTP timeout must be positive")
	}
	if len(messages) == 0 {
		return errors.New("at least one message is required")
	}
	for i, msg := range messages {
		if msg == nil {
			return fmt.Errorf("message %d must not be nil", i)
		}
		if err := msg.Validate(); err != nil {
			return fmt.Errorf("message %d is invalid: %w", i, err)
		}
	}

	switch c.encryption {
	case EncryptionImplicitTLS, EncryptionSTARTTLS:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidEncryption, c.encryption)
	}

	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))

	var auth smtp.Auth
	if c.username != "" || c.password != "" {
		auth = smtp.PlainAuth("", c.username, c.password, c.host)
	}

	if c.encryption == EncryptionImplicitTLS {
		return c.sendImplicitTLS(addr, auth, messages...)
	}
	return c.sendSTARTTLS(addr, auth, messages...)
}

func (c *Client) sendSTARTTLS(addr string, auth smtp.Auth, messages ...*Message) error {
	for _, msg := range messages {
		conn, err := (&net.Dialer{Timeout: c.timeout}).Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("connect to SMTP server: %w", err)
		}
		if err = conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
			conn.Close()
			return fmt.Errorf("set SMTP deadline: %w", err)
		}
		client, err := smtp.NewClient(conn, c.host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("initialize SMTP client: %w", err)
		}

		if ok, _ := client.Extension("STARTTLS"); !ok {
			client.Close()
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err = client.StartTLS(c.tlsConfig()); err != nil {
			client.Close()
			return fmt.Errorf("start TLS: %w", err)
		}

		if err = c.sendMessage(client, auth, msg); err != nil {
			client.Close()
			return err
		}
		if err = client.Quit(); err != nil {
			return fmt.Errorf("quit SMTP session: %w", err)
		}
	}
	return nil
}

func (c *Client) sendImplicitTLS(addr string, auth smtp.Auth, messages ...*Message) error {
	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, c.tlsConfig())
	if err != nil {
		return fmt.Errorf("connect to SMTP server over TLS: %w", err)
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer client.Close()
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate with SMTP server: %w", err)
		}
	}

	for _, msg := range messages {
		if err = c.sendMessage(client, nil, msg); err != nil {
			return err
		}
	}
	if err = client.Quit(); err != nil {
		return fmt.Errorf("quit SMTP session: %w", err)
	}
	return nil
}

func (c *Client) tlsConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: c.host,
		// SP 800-52 Rev. 2: ECDHE with authenticated AES-GCM for TLS 1.2.
		// Go controls TLS 1.3 suites and further restricts negotiation in FIPS mode.
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}
}

func (c *Client) sendMessage(client *smtp.Client, auth smtp.Auth, msg *Message) error {
	from, recipients, err := msg.envelope()
	if err != nil {
		return err
	}
	msgBytes, err := msg.Bytes()
	if err != nil {
		return err
	}
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate with SMTP server: %w", err)
		}
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	for _, recipient := range recipients {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("set SMTP recipient: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP data writer: %w", err)
	}
	if _, err = w.Write(msgBytes); err != nil {
		_ = w.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close SMTP data writer: %w", err)
	}
	return nil
}
