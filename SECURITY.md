# Security Policy for QuantFlow Terminal

## Reporting a Vulnerability

If you discover a security vulnerability in QuantFlow Terminal, please report it privately. **Do not report security issues via public GitHub issues.**

### How to Report

Send details to the project maintainers via email or open a private security advisory on GitHub:

- **Private advisory**: https://github.com/shenyzw/quantflow/security/advisories/new
- **Email**: [shenyzw@users.noreply.github.com](mailto:shenyzw@users.noreply.github.com)

### What to Include

- Version of QuantFlow Terminal affected
- Description of the vulnerability and potential impact
- Steps to reproduce (proof of concept preferred)
- Your contact information for follow-up questions

## Response Process

1. **Acknowledgment** -- We will acknowledge receipt within 48 hours.
2. **Assessment** -- We will assess the severity and impact within 5 business days.
3. **Fix** -- A fix will be developed and tested. Timeline depends on severity.
4. **Release** -- A patched version will be released, and the vulnerability disclosed.

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| Latest  | :white_check_mark: |
| Older   | :x:                |

Only the latest release receives security patches. Users are encouraged to always use the latest version.

## Encryption

QuantFlow Terminal uses **AES-256-GCM** for encrypting sensitive data at rest, including:

- Broker API keys
- Exchange credentials
- API tokens

Credentials are encrypted before being written to the SQLite database and decrypted only in memory. The encryption key is derived from a machine-local secret and is never transmitted.

## Security Best Practices

- Keep your Go toolchain, Node.js, and Python versions up to date
- Use API keys with minimal required permissions (read-only where possible)
- Store the QuantFlow config directory (`~/.config/quantflow`) with restricted permissions (`chmod 700`)
- Report any `//go:embed` or filesystem path traversal concerns immediately

## Scope

The following areas are in scope for security reports:

- Go backend (Wails app, trading engine, broker adapters)
- Frontend data handling (API key storage in Pinia stores)
- gRPC communication with the Python sidecar
- SQLite data storage and encryption
- Credential management

**Out of scope:** Third-party dependencies, unless a vulnerability in a dependency affects QuantFlow in a novel way.
