# Security Policy

## Supported Versions

We actively support and provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :x:                |
| < 0.1   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue in xk6-milvus, please report it responsibly.

### How to Report

**Please do NOT create a public GitHub issue for security vulnerabilities.**

Instead, please report security vulnerabilities by:

1. **Email**: Send details to [security contact email]
2. **GitHub Security Advisory**: Use GitHub's [private security reporting feature](https://github.com/mmga-lab/xk6-milvus/security/advisories/new)

### What to Include

When reporting a vulnerability, please include:

- **Description** - Clear description of the vulnerability
- **Impact** - Potential impact and severity
- **Reproduction** - Step-by-step instructions to reproduce
- **Environment** - OS, Go version, k6 version, Milvus version
- **Suggested Fix** - If you have ideas on how to fix it (optional)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Assessment**: Within 7 days
- **Fix Timeline**: Varies by severity
  - Critical: < 7 days
  - High: < 14 days
  - Medium: < 30 days
  - Low: Next scheduled release

### Disclosure Policy

- We will work with you to understand and address the issue
- We will keep you informed of our progress
- Once fixed, we will coordinate disclosure timing
- We will credit you in release notes (unless you prefer to remain anonymous)

## Security Best Practices

When using xk6-milvus:

### Authentication

- **Always use authentication** when connecting to production Milvus instances
- **Rotate credentials regularly**
- **Use environment variables** for credentials, never hardcode them

```javascript
// Good - use environment variable
const token = __ENV.MILVUS_TOKEN || 'root:Milvus';

// Bad - hardcoded credentials
const token = 'mypassword123';  // Don't do this!
```

### Connection Security

- **Use TLS/SSL** for Milvus connections in production
- **Restrict network access** to Milvus servers
- **Avoid exposing Milvus ports** to the public internet

### Data Safety

- **Validate input data** before inserting into Milvus
- **Sanitize filter expressions** to prevent injection attacks
- **Limit collection permissions** appropriately
- **Monitor for unusual activity**

### Testing

- **Use separate Milvus instances** for testing and production
- **Use test data only** - avoid production data in tests
- **Clean up test collections** after testing

## Known Security Considerations

### k6 Script Execution

k6 scripts have access to:
- Network connections
- File system (limited)
- Environment variables

Be cautious when:
- Running scripts from untrusted sources
- Sharing scripts that contain credentials
- Using dynamic imports

### Milvus SDK

This extension wraps the official Milvus Go SDK. Security issues in the underlying SDK may affect this extension.

Monitor:
- [Milvus Security Advisories](https://github.com/milvus-io/milvus/security)
- [Go SDK Security Updates](https://github.com/milvus-io/milvus/tree/master/client)

## Security Updates

Security fixes will be:
- Released as soon as possible
- Documented in CHANGELOG.md
- Announced in GitHub releases
- Tagged with severity level

## Questions?

If you have security questions (not reporting a vulnerability):
- Open a GitHub Discussion
- Check our documentation
- Review Milvus security docs

Thank you for helping keep xk6-milvus secure!
