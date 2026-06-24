# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

- **Email**: tech@as214745.ru
- **GitHub**: Open a private security advisory via the "Security" tab

Do NOT open a public issue for security vulnerabilities.

Response time: within 48 hours.

## Scope

- API injection via crafted IP/ASN/CIDR inputs
- Rate limiter bypass
- Cache poisoning
- Information disclosure through error messages
- Resource exhaustion (DoS via batch requests, etc.)

## Out of Scope

- Third-party source API keys leakage (config responsibility)
- Infrastructure-level vulnerabilities (K3s cluster config)
