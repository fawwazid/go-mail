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
	client := NewClient("smtp.example.com", 25, "", "", EncryptionType("NONE"))
	if err := client.Send(validTestMessage()); !errors.Is(err, ErrInvalidEncryption) {
		t.Fatalf("Send() error = %v, want ErrInvalidEncryption", err)
	}
}

func TestClient_RejectsUnknownEncryption(t *testing.T) {
	client := NewClient("smtp.example.com", 25, "", "", EncryptionType("TYPO"))
	if err := client.Send(validTestMessage()); !errors.Is(err, ErrInvalidEncryption) {
		t.Fatalf("Send() error = %v, want ErrInvalidEncryption", err)
	}
}

func TestClient_RejectsRemovedEncryptionNames(t *testing.T) {
	for _, mode := range []EncryptionType{"SSL", "TLS", "NONE"} {
		client := NewClient("smtp.example.com", 465, "", "", mode)
		if err := client.Send(validTestMessage()); !errors.Is(err, ErrInvalidEncryption) {
			t.Fatalf("mode %q: %v", mode, err)
		}
	}
}

func TestClient_ValidatesArgumentsBeforeDialing(t *testing.T) {
	tests := []struct {
		name   string
		client *Client
		msgs   []*Message
	}{
		{name: "nil client", client: nil, msgs: []*Message{validTestMessage()}},
		{name: "missing host", client: NewClient("", 465, "", "", EncryptionImplicitTLS), msgs: []*Message{validTestMessage()}},
		{name: "invalid port", client: NewClient("smtp.example.com", 0, "", "", EncryptionImplicitTLS), msgs: []*Message{validTestMessage()}},
		{name: "invalid timeout", client: NewClient("smtp.example.com", 465, "", "", EncryptionImplicitTLS, WithTimeout(0)), msgs: []*Message{validTestMessage()}},
		{name: "missing message", client: NewClient("smtp.example.com", 465, "", "", EncryptionImplicitTLS)},
		{name: "nil message", client: NewClient("smtp.example.com", 465, "", "", EncryptionImplicitTLS), msgs: []*Message{nil}},
		{name: "invalid message", client: NewClient("smtp.invalid", 465, "", "", EncryptionImplicitTLS), msgs: []*Message{NewMessage()}},
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
	client := NewClient("smtp.example.com", 465, "", "", EncryptionImplicitTLS, WithTimeout(5*time.Second))
	if client.timeout != 5*time.Second {
		t.Fatalf("timeout = %s", client.timeout)
	}
}

func TestMockClientCompatibilityAndMethods(t *testing.T) {
	wantErr := errors.New("mock send failure")
	client := &MockClient{Err: wantErr}
	message := validTestMessage()
	if err := client.Send(message); !errors.Is(err, wantErr) {
		t.Fatalf("Send() error = %v, want %v", err, wantErr)
	}
	if len(client.SentMessages) != 0 {
		t.Fatal("Send() recorded a message when configured to fail")
	}

	client.SetError(nil)
	if err := client.Send(message); err != nil {
		t.Fatal(err)
	}
	if len(client.SentMessages) != 1 || client.SentMessages[0] != message {
		t.Fatal("legacy SentMessages field did not record the message")
	}

	snapshot := client.Messages()
	if len(snapshot) != 1 || snapshot[0] != message {
		t.Fatal("Messages() returned an incorrect snapshot")
	}
	snapshot[0] = nil
	if client.Messages()[0] != message {
		t.Fatal("mutating the returned slice changed the recorded message list")
	}
}

func TestMockClientSetError(t *testing.T) {
	client := NewMockClient()
	wantErr := errors.New("mock send failure")
	client.SetError(wantErr)
	if err := client.Send(validTestMessage()); !errors.Is(err, wantErr) {
		t.Fatalf("Send() error = %v, want %v", err, wantErr)
	}
}
