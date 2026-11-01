# Changelog

## [2.0.1](https://github.com/fawwazid/go-mail/compare/v2.0.0...v2.0.1) (2026-11-01)


### Bug Fixes

* **release:** configure semantic versioning ([#8](https://github.com/fawwazid/go-mail/issues/8)) ([66c3380](https://github.com/fawwazid/go-mail/commit/66c33803434a729483e83ea29569ea54d78900ad))


### Chores

* configure Google Release Please ([ac27866](https://github.com/fawwazid/go-mail/commit/ac27866f05dedfc8bd2d40a537a213332d3995d4))
* reset Release Please baseline ([6a3634a](https://github.com/fawwazid/go-mail/commit/6a3634aebf214efe94bef0372ade5b35bcd3698c))
* sync agent skill setup ([46a0dab](https://github.com/fawwazid/go-mail/commit/46a0dab7cacdd8720e337da236f77c8a4e25ea41))


### Security

* bound mail recipients and attachments ([#7](https://github.com/fawwazid/go-mail/issues/7)) ([fd24dd1](https://github.com/fawwazid/go-mail/commit/fd24dd152e9af9b8cd2c5a97c721a04f8a529a02))

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
