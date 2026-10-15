package mail

import (
	"errors"
	"testing"
	"time"
)

func validTestMessage() *Message {
	return NewMessage().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetBody("text/plain", "body")
}

func TestClient_RejectsInsecureConnection(t *testing.T) {
	client := NewClient("smtp.example.com", 25, "", "", EncryptionNone)
	if err := client.Send(validTestMessage()); !errors.Is(err, ErrInsecureConnection) {
		t.Fatalf("Send() error = %v, want ErrInsecureConnection", err)
	}
}

func TestClient_RejectsUnknownEncryption(t *testing.T) {
	client := NewClient("smtp.example.com", 25, "", "", EncryptionType("TYPO"))
	if err := client.Send(validTestMessage()); !errors.Is(err, ErrInvalidEncryption) {
		t.Fatalf("Send() error = %v, want ErrInvalidEncryption", err)
	}
}

func TestClient_ValidatesArgumentsBeforeDialing(t *testing.T) {
	tests := []struct {
		name   string
		client *Client
		msgs   []*Message
	}{
		{name: "nil client", client: nil, msgs: []*Message{validTestMessage()}},
		{name: "missing host", client: NewClient("", 465, "", "", EncryptionSSL), msgs: []*Message{validTestMessage()}},
		{name: "invalid port", client: NewClient("smtp.example.com", 0, "", "", EncryptionSSL), msgs: []*Message{validTestMessage()}},
		{name: "invalid timeout", client: NewClient("smtp.example.com", 465, "", "", EncryptionSSL, WithTimeout(0)), msgs: []*Message{validTestMessage()}},
		{name: "missing message", client: NewClient("smtp.example.com", 465, "", "", EncryptionSSL)},
		{name: "nil message", client: NewClient("smtp.example.com", 465, "", "", EncryptionSSL), msgs: []*Message{nil}},
		{name: "invalid message", client: NewClient("smtp.invalid", 465, "", "", EncryptionSSL), msgs: []*Message{NewMessage()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.client.Send(tt.msgs...); err == nil {
				t.Fatal("Send() expected an error")
			}
		})
	}
}

func TestClient_WithTimeout(t *testing.T) {
	client := NewClient("smtp.example.com", 465, "", "", EncryptionSSL, WithTimeout(5*time.Second))
	if client.timeout != 5*time.Second {
		t.Fatalf("timeout = %s", client.timeout)
	}
}
