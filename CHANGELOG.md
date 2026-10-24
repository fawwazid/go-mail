# Changelog

## v2.0.0 — 2026-10-24

Require explicit verified SMTP TLS and reject malformed MIME headers.

### Breaking changes

- Module path is now `github.com/fawwazid/go-mail/v2`. Update imports and use Go 1.27.3 or later.
- Replace EncryptionSSL/EncryptionTLS/EncryptionNone with EncryptionImplicitTLS and EncryptionSTARTTLS.
- Accept only IMPLICIT_TLS or STARTTLS mode values; remove ErrInsecureConnection.

### Security and maintenance

- Require TLS 1.2 or later; TLS 1.2 is limited to ECDHE/AES-GCM suites.
- Reject CR/LF in MIME types; retain certificate verification, mandatory STARTTLS, timeouts, and message-size limits.
- Remove redundant code, obsolete comments, and compatibility paths.
- Add regression tests, current README examples, security guidance, and CI for Linux and Windows.
- FIPS policy tests use the frozen Go module v1.0.0 in both approved and strict assessment modes. This library is not independently FIPS certified; consult SECURITY.md for deployment requirements.

### Install

```sh
go get github.com/fawwazid/go-mail/v2@v2.0.0
```
