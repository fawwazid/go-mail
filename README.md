# go-mail

Install: `go get github.com/fawwazid/go-mail/v2@v2.0.0`.
See [v2 release notes](CHANGELOG.md) for breaking changes.

SMTP message building and sending over verified TLS. Requires Go 1.27.3.

## Usage

```go
package main

import (
 "log"
 mail "github.com/fawwazid/go-mail/v2"
)

func main() {
 message := mail.NewMessage().
  SetFrom("sender@example.com").
  AddTo("recipient@example.com").
  SetSubject("Example").
  SetBody("text/plain", "Message body")

 client := mail.NewClient("smtp.example.com", 587, "username", "password", mail.EncryptionSTARTTLS)
 if err := client.Send(message); err != nil { log.Fatal(err) }
}
```

Load real credentials from your application's secret configuration.

## API and transport

- `EncryptionSTARTTLS`: require STARTTLS before credentials or message data,
  usually on port 587.
- `EncryptionImplicitTLS`: establish TLS before SMTP, usually on port 465.
- `WithTimeout`: bound connection, handshake, and SMTP operations.
- `Message`: sender, To/Cc/Bcc, subject, text/HTML body, and attachments.
- `AddAttachment` reads a bounded file; `AddAttachmentData` copies supplied data.
- `Validate` and `Bytes` report malformed addresses, headers, and MIME types.
- `MockClient` supports application tests without sending email.

TLS 1.2 uses ECDHE with AES-GCM; Go manages TLS 1.3 and FIPS negotiation.
Certificate and hostname verification remain enabled. Combined unencoded
message data is limited to 25 MiB. Bcc recipients are omitted from message headers.

## Breaking changes

`EncryptionNone`, `EncryptionSSL`, `EncryptionTLS`, and
`ErrInsecureConnection` have been removed. The only mode values are
`IMPLICIT_TLS` and `STARTTLS`; old string values and unknown modes return
`ErrInvalidEncryption`. There is no plaintext or opportunistic TLS fallback.

See [SECURITY.md](SECURITY.md). Tests use local TLS peers, not a live provider.
Run `go test ./...` and `go vet ./...`. License: [MIT](LICENSE).
