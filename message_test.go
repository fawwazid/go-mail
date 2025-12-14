package mail

import (
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
	if !strings.Contains(s, "filename=\"test.txt\"") {
		t.Errorf("Attachment filename missing")
	}
	// Base64 of "file content" is "ZmlsZSBjb250ZW50"
	if !strings.Contains(s, "ZmlsZSBjb250ZW50") {
		t.Errorf("Attachment content missing")
	}
}
