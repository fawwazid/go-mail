package mail

import (
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

func TestMessage_Bytes(t *testing.T) {
	m := NewMessage()
	m.SetFrom("sender@example.com")
	m.AddTo("recipient@example.com")
	m.SetSubject("Test Subject")
	m.SetBody("text/plain", "Hello World")

	raw, err := m.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}

	s := string(raw)
	if !strings.Contains(s, "From: sender@example.com") {
		t.Errorf("From header missing")
	}
	if !strings.Contains(s, "To: recipient@example.com") {
		t.Errorf("To header missing")
	}
	if !strings.Contains(s, "Subject: Test Subject") {
		t.Errorf("Subject header missing")
	}
	if !strings.Contains(s, "Hello World") {
		t.Errorf("Body missing")
	}
}

func TestMessage_UnicodeSubject(t *testing.T) {
	m := NewMessage().
		SetFrom("Sender <sender@example.com>").
		AddTo("Recipient <recipient@example.com>").
		SetSubject("Laporan bulanan ✓").
		SetBody("text/plain", "Hello")

	raw, err := m.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}
	parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("DecodeHeader() error = %v", err)
	}
	if subject != "Laporan bulanan ✓" {
		t.Fatalf("Subject = %q", subject)
	}
}

func TestMessage_Envelope(t *testing.T) {
	m := NewMessage().
		SetFrom("Sender <sender@example.com>").
		AddCc("Copy <copy@example.com>").
		AddBcc("hidden@example.com")

	from, recipients, err := m.envelope()
	if err != nil {
		t.Fatalf("envelope() error = %v", err)
	}
	if from != "sender@example.com" {
		t.Errorf("from = %q", from)
	}
	if strings.Join(recipients, ",") != "copy@example.com,hidden@example.com" {
		t.Errorf("recipients = %v", recipients)
	}
}

func TestMessage_RejectsHeaderInjection(t *testing.T) {
	m := NewMessage().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("safe\r\nBcc: attacker@example.com")

	if _, err := m.Bytes(); err == nil {
		t.Fatal("Bytes() expected an error")
	}
}

func TestMessage_Attachments(t *testing.T) {
	m := NewMessage()
	m.SetFrom("sender@example.com")
	m.AddTo("recipient@example.com")
	m.SetSubject("Attachment Test")
	m.SetBody("text/plain", "See attachment")

	m.AddAttachmentData("test.txt", []byte("file content"), "text/plain")

	raw, err := m.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}

	s := string(raw)
	if !strings.Contains(s, "Content-Type: multipart/mixed") {
		t.Errorf("Expected multipart/mixed")
	}
	if !strings.Contains(s, "filename=test.txt") {
		t.Errorf("Attachment filename missing")
	}
	// Base64 of "file content" is "ZmlsZSBjb250ZW50"
	if !strings.Contains(s, "ZmlsZSBjb250ZW50") {
		t.Errorf("Attachment content missing")
	}
}

func TestMessage_AttachmentBase64LinesAreMIMECompliant(t *testing.T) {
	m := NewMessage().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetBody("text/plain", "body").
		AddAttachmentData("data.bin", make([]byte, 256), "application/octet-stream")

	raw, err := m.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error = %v", err)
	}
	parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	_, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType() error = %v", err)
	}
	reader := multipart.NewReader(parsed.Body, params["boundary"])
	if _, err = reader.NextPart(); err != nil {
		t.Fatalf("read body part: %v", err)
	}
	attachment, err := reader.NextPart()
	if err != nil {
		t.Fatalf("read attachment part: %v", err)
	}
	encoded, err := io.ReadAll(attachment)
	if err != nil {
		t.Fatalf("read attachment: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(encoded)), "\r\n") {
		if len(line) > 76 {
			t.Fatalf("base64 line has %d characters", len(line))
		}
	}
	decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(string(encoded))))
	if err != nil {
		t.Fatalf("decode attachment: %v", err)
	}
	if len(decoded) != 256 {
		t.Fatalf("decoded attachment size = %d", len(decoded))
	}
}

func TestMessage_AddAttachmentDataCopiesContent(t *testing.T) {
	content := []byte("original")
	m := NewMessage().AddAttachmentData("data.txt", content, "text/plain")
	content[0] = 'X'
	if string(m.Attachments[0].Content) != "original" {
		t.Fatal("attachment content changed after caller mutated its buffer")
	}
}

func TestMessage_RejectsOversizedBody(t *testing.T) {
	m := NewMessage().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetBody("text/plain", strings.Repeat("x", int(MaxMessageSize)+1))
	if err := m.Validate(); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("Validate() error = %v, want ErrMessageTooLarge", err)
	}
}

func TestMessage_RejectsInvalidContentType(t *testing.T) {
	m := NewMessage().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetBody("text/plain\r\nX-Injected: yes", "body")
	if err := m.Validate(); err == nil {
		t.Fatal("Validate() expected an error")
	}
}

func TestMessage_RejectsAddressHeaderInjection(t *testing.T) {
	tests := []*Message{
		NewMessage().SetFrom("sender@example.com\r\nBcc: attacker@example.com").AddTo("recipient@example.com"),
		NewMessage().SetFrom("sender@example.com").AddTo("recipient@example.com\r\nBcc: attacker@example.com"),
	}
	for _, message := range tests {
		if err := message.Validate(); err == nil {
			t.Fatal("Validate() expected an error")
		}
	}
}
