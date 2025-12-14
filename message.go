package mail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"
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
		Content:     content,
		ContentType: contentType,
	})
	return m
}

// Validate checks if the message has minimum required fields.
func (m *Message) Validate() error {
	if m.From == "" {
		return errors.New("from address is required")
	}
	if len(m.To) == 0 {
		return errors.New("at least one recipient (To) is required")
	}
	return nil
}

// Bytes returns the byte representation of the message.
func (m *Message) Bytes() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)

	// Headers
	buf.WriteString("From: " + m.From + "\r\n")
	if len(m.To) > 0 {
		buf.WriteString("To: " + strings.Join(m.To, ", ") + "\r\n")
	}
	if len(m.Cc) > 0 {
		buf.WriteString("Cc: " + strings.Join(m.Cc, ", ") + "\r\n")
	}
	buf.WriteString("Subject: " + m.Subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	if len(m.Attachments) > 0 {
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n")
		buf.WriteString("\r\n")

		// Body part
		bodyPartHeader := make(textproto.MIMEHeader)
		bodyPartHeader.Set("Content-Type", m.ContentType)
		bodyPart, err := writer.CreatePart(bodyPartHeader)
		if err != nil {
			return nil, err
		}
		bodyPart.Write([]byte(m.Body))

		// Attachments
		for _, att := range m.Attachments {
			attHeader := make(textproto.MIMEHeader)
			if att.Inline {
				attHeader.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", att.Filename))
			} else {
				attHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", att.Filename))
			}
			attHeader.Set("Content-Type", att.ContentType)
			attHeader.Set("Content-Transfer-Encoding", "base64")

			attPart, err := writer.CreatePart(attHeader)
			if err != nil {
				return nil, err
			}

			encoder := base64.NewEncoder(base64.StdEncoding, attPart)
			_, err := encoder.Write(att.Content)
			if err != nil {
				return nil, err
			}
			encoder.Close()
		}
		writer.Close()
	} else {
		// Simple message without attachments
		buf.WriteString("Content-Type: " + m.ContentType + "\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(m.Body)
	}

	return buf.Bytes(), nil
}
