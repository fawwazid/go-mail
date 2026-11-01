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

// MaxMessageSize is the maximum combined unencoded message size.
const MaxMessageSize int64 = 25 << 20

const (
	// MaxRecipients bounds the number of envelope recipients per message.
	MaxRecipients = 1000
	// MaxAttachments bounds the number of attachments per message.
	MaxAttachments             = 100
	rejectedAttachmentFilename = "\x00go-mail-rejected-attachment"
)

var (
	ErrMessageTooLarge        = errors.New("message exceeds maximum size")
	ErrTooManyRecipients      = errors.New("message exceeds maximum recipient count")
	ErrTooManyAttachments     = errors.New("message exceeds maximum attachment count")
	ErrAttachmentDataRejected = errors.New("attachment data was rejected; use AddAttachmentDataChecked for the cause")
)

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
		ContentType: "text/plain",
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
	m.addRecipients(&m.To, emails)
	return m
}

// AddCc adds recipients to the Cc list.
func (m *Message) AddCc(emails ...string) *Message {
	m.addRecipients(&m.Cc, emails)
	return m
}

// AddBcc adds recipients to the Bcc list.
func (m *Message) AddBcc(emails ...string) *Message {
	m.addRecipients(&m.Bcc, emails)
	return m
}

func (m *Message) addRecipients(target *[]string, values []string) {
	if m == nil || len(values) == 0 {
		return
	}
	count, overLimit := m.recipientCount()
	if overLimit {
		return
	}
	remaining := MaxRecipients - count
	if remaining == 0 {
		*target = append(*target, "")
		return
	}
	if len(values) > remaining {
		*target = append(*target, values[:remaining]...)
		// Keep a bounded invalid marker so Validate fails closed without copying
		// the rest of an oversized variadic input.
		*target = append(*target, "")
		return
	}
	*target = append(*target, values...)
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
	if m == nil {
		return errors.New("message must not be nil")
	}
	if len(m.Attachments) >= MaxAttachments {
		return ErrTooManyAttachments
	}
	if _, overLimit := m.recipientCount(); overLimit {
		return ErrTooManyRecipients
	}
	a, err := CreateAttachmentFromFile(path, opts...)
	if err != nil {
		return err
	}
	if _, err := m.addedAttachmentSize(a.Filename, a.ContentType, len(a.Content)); err != nil {
		return err
	}
	m.Attachments = append(m.Attachments, a)
	return nil
}

// AddAttachmentData adds an attachment from bytes.
func (m *Message) AddAttachmentData(filename string, content []byte, contentType string) *Message {
	if err := m.AddAttachmentDataChecked(filename, content, contentType); err != nil && m != nil {
		// Preserve the legacy fluent API while ensuring invalid data fails closed
		// when the message is validated or sent. The marker contains no caller data.
		if !m.hasRejectedAttachment() {
			m.Attachments = append(m.Attachments, &Attachment{Filename: rejectedAttachmentFilename})
		}
	}
	return m
}

// AddAttachmentDataChecked adds an attachment after checking message limits.
// It returns an error immediately and does not copy content when a limit is
// exceeded. Prefer this method when attachment input may be untrusted.
func (m *Message) AddAttachmentDataChecked(filename string, content []byte, contentType string) error {
	if m == nil {
		return errors.New("message must not be nil")
	}
	if len(m.Attachments) >= MaxAttachments {
		return ErrTooManyAttachments
	}
	if _, overLimit := m.recipientCount(); overLimit {
		return ErrTooManyRecipients
	}
	if int64(len(content)) > MaxMessageSize {
		return ErrMessageTooLarge
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if _, err := m.addedAttachmentSize(filename, contentType, len(content)); err != nil {
		return err
	}
	m.Attachments = append(m.Attachments, &Attachment{
		Filename:    filename,
		Content:     append([]byte(nil), content...),
		ContentType: contentType,
	})
	return nil
}

// Validate checks if the message has minimum required fields.
func (m *Message) Validate() error {
	if m == nil {
		return errors.New("message must not be nil")
	}
	if m.hasRejectedAttachment() {
		return ErrAttachmentDataRejected
	}
	if len(m.Attachments) > MaxAttachments {
		return ErrTooManyAttachments
	}
	if _, overLimit := m.recipientCount(); overLimit {
		return ErrTooManyRecipients
	}
	if _, err := m.messageSize(); err != nil {
		return err
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
	for _, recipients := range [][]string{m.To, m.Cc, m.Bcc} {
		for _, value := range recipients {
			if _, err := mail.ParseAddress(value); err != nil {
				return fmt.Errorf("invalid recipient address %q: %w", value, err)
			}
		}
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("subject must not contain line breaks")
	}
	if strings.ContainsAny(m.ContentType, "\r\n") {
		return errors.New("content type must not contain line breaks")
	}
	if _, _, err := mime.ParseMediaType(m.ContentType); err != nil {
		return fmt.Errorf("invalid content type %q: %w", m.ContentType, err)
	}
	for _, attachment := range m.Attachments {
		if attachment == nil {
			return errors.New("attachment must not be nil")
		}
		if attachment.Filename == "" {
			return errors.New("attachment filename is required")
		}
		if strings.ContainsAny(attachment.ContentType, "\r\n") {
			return errors.New("attachment content type must not contain line breaks")
		}
		if _, _, err := mime.ParseMediaType(attachment.ContentType); err != nil {
			return fmt.Errorf("invalid attachment content type %q: %w", attachment.ContentType, err)
		}
	}
	return nil
}

func (m *Message) messageSize() (int64, error) {
	total := int64(0)
	add := func(size int) error {
		if int64(size) > MaxMessageSize-total {
			return fmt.Errorf("%w: maximum is %d bytes", ErrMessageTooLarge, MaxMessageSize)
		}
		total += int64(size)
		return nil
	}
	for _, size := range []int{len(m.From), len(m.Subject), len(m.Body), len(m.ContentType)} {
		if err := add(size); err != nil {
			return 0, err
		}
	}
	for _, recipients := range [][]string{m.To, m.Cc, m.Bcc} {
		for _, recipient := range recipients {
			if err := add(len(recipient)); err != nil {
				return 0, err
			}
		}
	}
	for _, attachment := range m.Attachments {
		if attachment == nil {
			return 0, errors.New("attachment must not be nil")
		}
		for _, size := range []int{len(attachment.Filename), len(attachment.ContentType), len(attachment.Content)} {
			if err := add(size); err != nil {
				return 0, err
			}
		}
	}
	return total, nil
}

func (m *Message) recipientCount() (int, bool) {
	count := 0
	for _, recipients := range [][]string{m.To, m.Cc, m.Bcc} {
		if len(recipients) > MaxRecipients-count {
			return MaxRecipients + 1, true
		}
		count += len(recipients)
	}
	return count, false
}

func (m *Message) addedAttachmentSize(filename, contentType string, contentLength int) (int64, error) {
	current, err := m.messageSize()
	if err != nil {
		return 0, err
	}
	for _, size := range []int{len(filename), len(contentType), contentLength} {
		if int64(size) > MaxMessageSize-current {
			return 0, fmt.Errorf("%w: maximum is %d bytes", ErrMessageTooLarge, MaxMessageSize)
		}
		current += int64(size)
	}
	return current, nil
}

func (m *Message) hasRejectedAttachment() bool {
	for _, attachment := range m.Attachments {
		if attachment != nil && attachment.Filename == rejectedAttachmentFilename {
			return true
		}
	}
	return false
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
