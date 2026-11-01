package mail

import (
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
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

func TestMessage_AddAttachmentDataCheckedRejectsBeforeCopy(t *testing.T) {
	content := make([]byte, int(MaxMessageSize)+1)
	m := NewMessage()
	if err := m.AddAttachmentDataChecked("large.bin", content, "application/octet-stream"); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("AddAttachmentDataChecked() error = %v, want ErrMessageTooLarge", err)
	}
	if len(m.Attachments) != 0 {
		t.Fatalf("rejected attachment was retained: %d attachments", len(m.Attachments))
	}
}

func TestMessage_AddAttachmentDataLegacyFailsClosed(t *testing.T) {
	content := make([]byte, int(MaxMessageSize)+1)
	m := NewMessage().AddAttachmentData("large.bin", content, "application/octet-stream")
	if len(m.Attachments) != 1 || len(m.Attachments[0].Content) != 0 {
		t.Fatal("legacy method retained or copied rejected attachment data")
	}
	if err := m.Validate(); !errors.Is(err, ErrAttachmentDataRejected) {
		t.Fatalf("Validate() error = %v, want ErrAttachmentDataRejected", err)
	}
}

func TestMessage_AddAttachmentDataCheckedRejectsAggregateOverflow(t *testing.T) {
	m := NewMessage().SetBody("text/plain", strings.Repeat("x", int(MaxMessageSize)))
	if err := m.AddAttachmentDataChecked("a.bin", []byte{1}, "application/octet-stream"); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("AddAttachmentDataChecked() error = %v, want ErrMessageTooLarge", err)
	}
	if len(m.Attachments) != 0 {
		t.Fatal("aggregate-overflow attachment was retained")
	}
}

func TestMessage_RejectsExcessiveRecipientsBeforeParsing(t *testing.T) {
	m := NewMessage().SetFrom("sender@example.com")
	m.To = make([]string, MaxRecipients+1)
	if err := m.Validate(); !errors.Is(err, ErrTooManyRecipients) {
		t.Fatalf("Validate() error = %v, want ErrTooManyRecipients", err)
	}

	m.To = make([]string, MaxRecipients)
	m.Cc = []string{"recipient@example.com"}
	if err := m.Validate(); !errors.Is(err, ErrTooManyRecipients) {
		t.Fatalf("Validate() combined count error = %v, want ErrTooManyRecipients", err)
	}
}

func TestMessage_AddRecipientBuildersBoundVariadicInput(t *testing.T) {
	values := make([]string, MaxRecipients+10_000)
	m := NewMessage().AddTo(values...)
	if len(m.To) != MaxRecipients+1 {
		t.Fatalf("AddTo retained %d recipients, want bounded marker count %d", len(m.To), MaxRecipients+1)
	}
	if err := m.Validate(); !errors.Is(err, ErrTooManyRecipients) {
		t.Fatalf("Validate() error = %v, want ErrTooManyRecipients", err)
	}
}

func TestMessage_RejectsExcessiveAttachments(t *testing.T) {
	m := NewMessage()
	m.Attachments = make([]*Attachment, MaxAttachments+1)
	if err := m.AddAttachmentDataChecked("extra.bin", []byte{1}, "application/octet-stream"); !errors.Is(err, ErrTooManyAttachments) {
		t.Fatalf("AddAttachmentDataChecked() error = %v, want ErrTooManyAttachments", err)
	}
	if err := m.Validate(); !errors.Is(err, ErrTooManyAttachments) {
		t.Fatalf("Validate() error = %v, want ErrTooManyAttachments", err)
	}
}

func TestMessage_AddAttachmentRejectsLimits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(path, []byte("small"), 0o600); err != nil {
		t.Fatal(err)
	}

	tooMany := NewMessage()
	tooMany.Attachments = make([]*Attachment, MaxAttachments)
	if err := tooMany.AddAttachment(filepath.Join(dir, "missing.txt")); !errors.Is(err, ErrTooManyAttachments) {
		t.Fatalf("AddAttachment() count error = %v, want ErrTooManyAttachments", err)
	}

	tooLarge := NewMessage().SetBody("text/plain", strings.Repeat("x", int(MaxMessageSize)))
	if err := tooLarge.AddAttachment(path); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("AddAttachment() size error = %v, want ErrMessageTooLarge", err)
	}
	if len(tooLarge.Attachments) != 0 {
		t.Fatal("aggregate-overflow file attachment was retained")
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
