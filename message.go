package mail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
)

// MaxMessageSize is the maximum combined unencoded body and attachment size.
const MaxMessageSize int64 = 25 << 20

var ErrMessageTooLarge = errors.New("message exceeds maximum size")

// Message represents an email message.
type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	ContentType string // "text/plain" or "text/html"
	Attachments []*Attachment
}

// NewMessage creates a new empty message.
// Default Content-Type is text/plain.
func NewMessage() *Message {
	return &Message{
		ContentType: "text/plain", // Default to plain text
		To:          make([]string, 0),
		Cc:          make([]string, 0),
		Bcc:         make([]string, 0),
		Attachments: make([]*Attachment, 0),
	}
}

// SetFrom sets the sender address.
func (m *Message) SetFrom(from string) *Message {
	m.From = from
	return m
}

// AddTo adds recipients to the To list.
func (m *Message) AddTo(emails ...string) *Message {
	m.To = append(m.To, emails...)
	return m
}

// AddCc adds recipients to the Cc list.
func (m *Message) AddCc(emails ...string) *Message {
	m.Cc = append(m.Cc, emails...)
	return m
}

// AddBcc adds recipients to the Bcc list.
func (m *Message) AddBcc(emails ...string) *Message {
	m.Bcc = append(m.Bcc, emails...)
	return m
}

// SetSubject sets the email subject.
func (m *Message) SetSubject(subject string) *Message {
	m.Subject = subject
	return m
}

// SetBody sets the email body and content type.
func (m *Message) SetBody(contentType, body string) *Message {
	m.ContentType = contentType
	m.Body = body
	return m
}

// AddAttachment adds an attachment from a file path.
func (m *Message) AddAttachment(path string, opts ...FileOption) error {
	a, err := CreateAttachmentFromFile(path, opts...)
	if err != nil {
		return err
	}
	m.Attachments = append(m.Attachments, a)
	return nil
}

// AddAttachmentData adds an attachment from bytes.
func (m *Message) AddAttachmentData(filename string, content []byte, contentType string) *Message {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	m.Attachments = append(m.Attachments, &Attachment{
		Filename:    filename,
		Content:     append([]byte(nil), content...),
		ContentType: contentType,
	})
	return m
}

// Validate checks if the message has minimum required fields.
func (m *Message) Validate() error {
	if m == nil {
		return errors.New("message must not be nil")
	}
	if m.From == "" {
		return errors.New("from address is required")
	}
	if _, err := mail.ParseAddress(m.From); err != nil {
		return fmt.Errorf("invalid from address %q: %w", m.From, err)
	}
	if len(m.To)+len(m.Cc)+len(m.Bcc) == 0 {
		return errors.New("at least one recipient is required")
	}
	for _, value := range append(append(append([]string(nil), m.To...), m.Cc...), m.Bcc...) {
		if _, err := mail.ParseAddress(value); err != nil {
			return fmt.Errorf("invalid recipient address %q: %w", value, err)
		}
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("subject must not contain line breaks")
	}
	if _, _, err := mime.ParseMediaType(m.ContentType); err != nil {
		return fmt.Errorf("invalid content type %q: %w", m.ContentType, err)
	}
	totalSize := int64(0)
	addSize := func(size int) error {
		if int64(size) > MaxMessageSize-totalSize {
			return fmt.Errorf("%w: maximum is %d bytes", ErrMessageTooLarge, MaxMessageSize)
		}
		totalSize += int64(size)
		return nil
	}
	if err := addSize(len(m.From) + len(m.Subject) + len(m.Body) + len(m.ContentType)); err != nil {
		return err
	}
	for _, value := range append(append(append([]string(nil), m.To...), m.Cc...), m.Bcc...) {
		if err := addSize(len(value)); err != nil {
			return err
		}
	}
	for _, attachment := range m.Attachments {
		if attachment == nil {
			return errors.New("attachment must not be nil")
		}
		if attachment.Filename == "" {
			return errors.New("attachment filename is required")
		}
		if _, _, err := mime.ParseMediaType(attachment.ContentType); err != nil {
			return fmt.Errorf("invalid attachment content type %q: %w", attachment.ContentType, err)
		}
		if err := addSize(len(attachment.Filename) + len(attachment.ContentType)); err != nil {
			return err
		}
		if err := addSize(len(attachment.Content)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Message) envelope() (string, []string, error) {
	if err := m.Validate(); err != nil {
		return "", nil, err
	}
	from, _ := mail.ParseAddress(m.From)
	values := append(append(append([]string(nil), m.To...), m.Cc...), m.Bcc...)
	recipients := make([]string, 0, len(values))
	for _, value := range values {
		address, _ := mail.ParseAddress(value)
		recipients = append(recipients, address.Address)
	}
	return from.Address, recipients, nil
}

// Bytes returns the byte representation of the message.
func (m *Message) Bytes() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)

	// Headers
	buf.WriteString("From: " + formatAddress(m.From) + "\r\n")
	if len(m.To) > 0 {
		buf.WriteString("To: " + formatAddresses(m.To) + "\r\n")
	}
	if len(m.Cc) > 0 {
		buf.WriteString("Cc: " + formatAddresses(m.Cc) + "\r\n")
	}
	buf.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", m.Subject) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(m.Attachments) > 0 {
		writer := multipart.NewWriter(buf)
		buf.WriteString("Content-Type: " + mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": writer.Boundary()}) + "\r\n")
		buf.WriteString("\r\n")

		// Body part
		bodyPartHeader := make(textproto.MIMEHeader)
		bodyPartHeader.Set("Content-Type", m.ContentType)
		bodyPartHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		bodyPart, err := writer.CreatePart(bodyPartHeader)
		if err != nil {
			return nil, err
		}
		bodyEncoder := quotedprintable.NewWriter(bodyPart)
		if _, err = bodyEncoder.Write([]byte(m.Body)); err != nil {
			return nil, err
		}
		if err = bodyEncoder.Close(); err != nil {
			return nil, err
		}

		// Attachments
		for _, att := range m.Attachments {
			attHeader := make(textproto.MIMEHeader)
			disposition := "attachment"
			if att.Inline {
				disposition = "inline"
			}
			attHeader.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": att.Filename}))
			attHeader.Set("Content-Type", att.ContentType)
			attHeader.Set("Content-Transfer-Encoding", "base64")

			attPart, err := writer.CreatePart(attHeader)
			if err != nil {
				return nil, err
			}

			encoder := base64.NewEncoder(base64.StdEncoding, &mimeLineWriter{w: attPart})
			_, err = encoder.Write(att.Content)
			if err != nil {
				return nil, err
			}
			if err = encoder.Close(); err != nil {
				return nil, err
			}
		}
		if err = writer.Close(); err != nil {
			return nil, err
		}
	} else {
		buf.WriteString("Content-Type: " + m.ContentType + "\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		bodyEncoder := quotedprintable.NewWriter(buf)
		if _, err := bodyEncoder.Write([]byte(m.Body)); err != nil {
			return nil, err
		}
		if err := bodyEncoder.Close(); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func formatAddress(value string) string {
	address, _ := mail.ParseAddress(value)
	if address.Name == "" {
		return address.Address
	}
	return address.String()
}

func formatAddresses(values []string) string {
	formatted := make([]string, len(values))
	for i, value := range values {
		formatted[i] = formatAddress(value)
	}
	return strings.Join(formatted, ", ")
}

type mimeLineWriter struct {
	w      io.Writer
	column int
}

func (w *mimeLineWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		remaining := 76 - w.column
		if remaining == 0 {
			if _, err := io.WriteString(w.w, "\r\n"); err != nil {
				return written, err
			}
			w.column = 0
			remaining = 76
		}
		n := len(p)
		if n > remaining {
			n = remaining
		}
		count, err := w.w.Write(p[:n])
		written += count
		w.column += count
		p = p[count:]
		if err != nil {
			return written, err
		}
		if count != n {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}
