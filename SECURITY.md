# Security Policy

## Supported versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | ✅          |

## Reporting a vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Email: [tech@as214745.ru](mailto:tech@as214745.ru)
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

## Response timeline

- Acknowledgment within 48 hours
- Fix or mitigation within 7 days for critical issues
- Credit in release notes (unless you prefer anonymity)

## Security measures

- Rate limiting per IP with configurable limits
- Request body size limits (1MB for batch endpoint)
- X-Forwarded-For sanitization
- Input validation on all endpoints
- No secrets in code — all config via environment variables
