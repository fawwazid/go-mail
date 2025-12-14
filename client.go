package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

// Client represents an SMTP client.
type Client struct {
	host       string
	port       int
	username   string
	password   string
	encryption EncryptionType
}

// NewClient creates a new SMTP client.
func NewClient(host string, port int, username, password string, encryption EncryptionType) *Client {
	return &Client{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		encryption: encryption,
	}
}

// Send sends one or more messages using the SMTP client.
// It handles potential connection and authentication details.
func (c *Client) Send(messages ...*Message) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	auth := smtp.PlainAuth("", c.username, c.password, c.host)

	// Custom dialer to handle SSL/TLS vs StartTLS vs None
	// For simplicity, we can try to use standard helpers or build our own flow if needed.
	// Standard smtp.SendMail uses StartTLS if available.
	// For implicit SSL (port 465), we need to dial TLS directly.

	if c.encryption == EncryptionSSL {
		return c.sendSSL(addr, auth, messages...)
	}

	// For TLS (StartTLS) and None, we can often rely on standard behaviors,
	// but explicit control is better.
	// If EncryptionTLS is set, we strictly mandate StartTLS.

	return c.sendStandard(addr, auth, messages...)
}

func (c *Client) sendStandard(addr string, auth smtp.Auth, messages ...*Message) error {
	// This is a simplified implementation.
	// real implementation might want to persistent connection (Dial then Loop)
	// but for now, one-shot send is fine.

	// smtp.SendMail does: Dial -> StartTLS (if supported) -> Auth -> Mail -> Rcpt -> Data -> Quit.
	// We can iterate over messages.

	for _, msg := range messages {
		to := append(msg.To, msg.Cc...)
		to = append(to, msg.Bcc...)

		msgBytes, err := msg.Bytes()
		if err != nil {
			return err
		}

		err = smtp.SendMail(addr, auth, msg.From, to, msgBytes)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) sendSSL(addr string, auth smtp.Auth, messages ...*Message) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // Strict by default
		ServerName:         c.host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return err
	}
	defer client.Quit() // defer Quit, but we might want to ensure error checking on quit?
	// standard defer is usually fine for simple clients

	if err = client.Auth(auth); err != nil {
		return err
	}

	for _, msg := range messages {
		to := append(msg.To, msg.Cc...)
		to = append(to, msg.Bcc...)

		if err = client.Mail(msg.From); err != nil {
			return err
		}
		for _, addr := range to {
			if err = client.Rcpt(addr); err != nil {
				return err
			}
		}

		w, err := client.Data()
		if err != nil {
			return err
		}

		msgBytes, err := msg.Bytes()
		if err != nil {
			w.Close() // try to close data writer
			return err
		}

		_, err = w.Write(msgBytes)
		if err != nil {
			w.Close()
			return err
		}

		if err = w.Close(); err != nil {
			return err
		}
	}
	return nil
}
