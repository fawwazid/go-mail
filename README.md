# go-mail

[![Go Reference](https://pkg.go.dev/badge/github.com/fawwazid/go-mail.svg)](https://pkg.go.dev/github.com/fawwazid/go-mail)
[![Go Report Card](https://goreportcard.com/badge/github.com/fawwazid/go-mail)](https://goreportcard.com/report/github.com/fawwazid/go-mail)

`go-mail`: Go library for sending emails, supporting SMTP, attachments, and convenient message building.

## Installation

```bash
go get github.com/fawwazid/go-mail
```

## Features

- **Fluent Message Builder**: Easily construct emails with a chainable API.
- **Attachments**: Support for file attachments and inline files, with automatic MIME detection and base64 encoding.
- **Secure SMTP**: Built-in support for implicit TLS and mandatory StartTLS, with TLS 1.2 as the minimum.
- **Safe Defaults**: Network timeouts, message-size limits, validated addresses, and MIME-safe encoding.
- **Mocking**: Includes a `MockClient` for easy unit testing in your application.

## Usage

### Sending a Simple Email

```go
package main

import (
	"log"

	"github.com/fawwazid/go-mail"
)

func main() {
	// Create a new message
	msg := mail.NewMessage().
		SetFrom("me@example.com").
		AddTo("you@example.com").
		SetSubject("Hello from Go!").
		SetBody("text/plain", "This is a test email sent using go-mail.")

	// Create client (e.g., using Mailtrap, Gmail, etc.)
	// For Gmail usage, you likely need an App Password.
	client := mail.NewClient("smtp.example.com", 587, "user", "pass", mail.EncryptionTLS)

	// Send the email
	if err := client.Send(msg); err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}
}
```

### Sending HTML Email with Attachments

```go
msg := mail.NewMessage().
    SetFrom("sender@example.com").
    AddTo("recipient@example.com").
    SetSubject("Monthly Report").
    SetBody("text/html", "<h1>Report</h1><p>Please find the report attached.</p>")

// Add file from disk
if err := msg.AddAttachment("./report.pdf"); err != nil {
    log.Fatal(err)
}

// Override the detected filename or MIME type when needed.
if err := msg.AddAttachment("./report", mail.WithFileName("report.pdf"), mail.WithContentType("application/pdf")); err != nil {
    log.Fatal(err)
}

// Add file from bytes
data := []byte("some content")
msg.AddAttachmentData("data.txt", data, "text/plain")

client := mail.NewClient("smtp.example.com", 465, "user", "pass", mail.EncryptionSSL)
if err := client.Send(msg); err != nil {
    log.Fatal(err)
}
```

The default SMTP timeout is 30 seconds. Override it when needed:

```go
client := mail.NewClient(
    "smtp.example.com",
    465,
    "user",
    "pass",
    mail.EncryptionSSL,
    mail.WithTimeout(10*time.Second),
)
```

Unencrypted SMTP (`EncryptionNone`) is retained only for source compatibility and is rejected by `Send`. The combined unencoded body and attachment size is limited by `mail.MaxMessageSize` (25 MiB).

## Testing

You can use `mail.MockClient` to test your code without sending real emails.

```go
mock := mail.NewMockClient()
myCodeThatSendsEmail(mock) // Inject the mock

if len(mock.SentMessages) != 1 {
    t.Errorf("Expected 1 message, got %d", len(mock.SentMessages))
}
```

## License

MIT
