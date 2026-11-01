# Agent instructions

This repository is the `go-mail` Go module. Keep changes within this module
unless the task explicitly spans other repositories. Preserve existing user
changes. Do not create or edit changelogs; Release Please generates them.

## Security and API

- Require implicit TLS or STARTTLS; never add plaintext fallback or disable
  certificate/hostname validation.
- Keep SMTP timeouts, message and attachment size bounds, and header/MIME
  injection checks intact.
- Limit messages to 1,000 recipients, 100 attachments, and 25 MiB combined
  unencoded data. Use `AddAttachmentDataChecked` for untrusted attachment bytes.
- Keep Bcc addresses out of serialized message headers.
- Treat the legacy exported `MockClient` fields as deprecated; prefer
  `Messages()` and `SetError()`. Direct field access is not concurrency-safe.
- Describe FIPS status as dependent on the Go module, approved mode, and
  operating environment; this package is not independently certified.

## Development

- Workspace mail invariant: `go-mail` caps messages at 1,000 recipients, 100
  attachments, and 25 MiB; untrusted attachment bytes use its checked API.
- Workspace MFA invariant: `ConstantTimeCompare` rejects strings longer than
  4,096 UTF-8 bytes before allocating byte slices.

- Target the Go version in `go.mod`; format Go changes with `gofmt`.
- Keep README and security documentation aligned with transport behavior and
  validation limits. Do not claim CI coverage unless workflows exist here.
- Remove only clearly unused internal code; preserve exported APIs unless a
  breaking change is explicitly requested.
- Add or update regression tests for every code change and run the affected
  module's complete test suite before finishing. Run `go vet ./...` as well.
- Review mail and TLS changes against applicable OWASP and NIST SP 800-52
  guidance; retain certificate verification, secure transport, and injection
  defenses.
- For every workspace code, test, or security-policy change, review and update
  all eight module-level `AGENTS.md` files in the same change set so their
  constraints and test-coverage notes stay aligned. Keep updates concise; this
  is not a changelog.

## Test coverage notes

- `client_test.go` covers legacy `MockClient` fields, `SetError`, and the
  `Messages()` slice snapshot behavior.
- `message_test.go` covers pre-copy attachment bounds, fail-closed legacy
  behavior, aggregate message size, bounded variadic recipients, recipient
  count, and attachment count.

## Release workflow

- Use Conventional Commit pull request titles so squash merges produce the
  intended SemVer bump: `fix:` for patch, `feat:` for minor, and `!` or
  `BREAKING CHANGE:` for major. Use `fix(security):` for security fixes;
  `security:` alone does not bump a version.
- `docs:`, `test:`, `chore:`, and `ci:` do not trigger a release. Release
  Please opens or updates a release pull request; merge that generated PR to
  create the version tag and GitHub release.
- Never create or edit changelogs manually; Release Please generates them.
- Keep the temporary `last-release-sha` at the v2 preparation commit during this rollout: the existing v2.0.0 tag is not on `main` history. After the first generated v2.0.1 release PR is merged and tagged, remove this fixed marker in a chore PR so later releases track the latest tag automatically.