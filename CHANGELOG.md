# Changelog

## [3.0.0](https://github.com/fawwazid/go-mail/compare/v2.0.0...v3.0.0) (2026-11-01)


### ⚠ BREAKING CHANGES

* prepare v2 security refactor ([#5](https://github.com/fawwazid/go-mail/issues/5))

### Features

* implement a Go mail package for sending emails with SMTP client and attachment support ([97d9bd1](https://github.com/fawwazid/go-mail/commit/97d9bd180ab2c24ec766aafc7b8eabb1aa162a66))
* implement a Go mail package for sending emails with SMTP client… ([d4e6e90](https://github.com/fawwazid/go-mail/commit/d4e6e90ec4575f55821a7cb6e8533e303f430d17))
* prepare v2 security refactor ([#5](https://github.com/fawwazid/go-mail/issues/5)) ([525d3d0](https://github.com/fawwazid/go-mail/commit/525d3d00a5584cf6d8141d3fb7a832c004224bf0))

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

## v1.0.0 — 2026-10-15

- Initial release of the SMTP mail package with attachment support and security hardening.
