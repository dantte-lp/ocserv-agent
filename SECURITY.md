# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.3.x   | :white_check_mark: |
| < 0.3   | :x:                |

## Reporting a Vulnerability

We take the security of ocserv-agent seriously. If you have discovered a security vulnerability, please report it to us responsibly.

### Where to Report

**Please DO NOT report security vulnerabilities through public GitHub issues.**

Instead, please report security vulnerabilities by:

1. **GitHub Security Advisories** (Preferred)
   - Navigate to the [Security tab](https://github.com/dantte-lp/ocserv-agent/security/advisories)
   - Click "Report a vulnerability"
   - Fill in the details

2. **Email**
   - Send details to: [security contact - to be configured]
   - Use PGP key if available: [PGP key fingerprint - to be added]

### What to Include

Please include the following information in your report:

- Type of vulnerability (e.g., remote code execution, authentication bypass, privilege escalation)
- Full paths of source file(s) related to the vulnerability
- Location of the affected source code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the vulnerability, including how an attacker might exploit it

### Response Timeline

- **Initial Response**: Within 48 hours of receiving the report
- **Status Update**: Within 7 days with assessment and estimated timeline
- **Fix Development**: Varies by severity (Critical: <7 days, High: <14 days, Medium: <30 days)
- **Public Disclosure**: Coordinated with reporter after fix is released

### Security Update Process

1. Security issue is reported and confirmed
2. Fix is developed in a private repository/branch
3. Security advisory is drafted (if warranted)
4. Patch is released with security advisory
5. Public disclosure via:
   - GitHub Security Advisories
   - Release notes
   - README security badge update

## Security Measures

### Built-in Security Features

- **mTLS Authentication**: Client certificate authentication required for all gRPC connections
- **TLS 1.3 Minimum**: Strong encryption with modern TLS protocol
- **Command Whitelist**: Only approved commands (occtl, systemctl) are allowed
- **Input Validation**: Protection against command injection and path traversal
- **Audit Logging**: All commands logged with context (user, timestamp, result)
- **Least Privilege**: Designed to run with minimal required capabilities

### Security Scanning

This project uses multiple security scanning tools:

- **gosec**: Go security checker for common vulnerabilities
- **govulncheck**: Official Go vulnerability database scanner
- **Trivy**: Container and dependency vulnerability scanner
- **CodeQL**: Static analysis security testing (SAST)
- **Dependabot**: Automated dependency updates

Security scans run on:
- Every pull request
- Every push to main branch
- Nightly scheduled scans

### Security Best Practices

When deploying ocserv-agent:

1. **TLS Certificates**
   - Use certificates from a trusted CA
   - Rotate certificates before expiration
   - See [docs/CERTIFICATES.md](docs/CERTIFICATES.md)

2. **Network Security**
   - Restrict agent port (default: 9090) to control server IPs only
   - Use firewall rules to limit access
   - Consider VPN/tunnel for control plane communication

3. **File Permissions**
   - Config file: `0600` (readable only by agent user)
   - Certificate keys: `0600` (readable only by agent user)
   - Binary: `0755` (executable by all, writable only by root)

4. **System Hardening**
   - Run agent as dedicated non-root user when possible
   - Use systemd service with security hardening options
   - Enable SELinux/AppArmor if available

5. **Monitoring**
   - Monitor agent logs for suspicious activity
   - Set up alerts for failed authentication attempts
   - Review audit logs regularly

## Known Security Considerations

### Architecture Security Model

ocserv-agent is designed to:
- Accept commands from a trusted control server
- Execute privileged operations (occtl, systemctl) on behalf of the control server
- Operate with elevated privileges for ocserv management

**This means:**
- The control server is a critical security component
- Compromise of control server credentials = compromise of all agents
- mTLS certificates must be protected as root-equivalent credentials

### Out of Scope

The following are considered out of scope for security reports:

- Denial of service attacks requiring excessive resources
- Security issues in third-party dependencies (report to upstream)
- Security issues in ocserv itself (report to [ocserv project](https://gitlab.com/openconnect/ocserv))
- Theoretical vulnerabilities without practical exploit
- Social engineering attacks

## Security Disclosure Policy

### Coordinated Disclosure

We follow coordinated disclosure principles:

1. Security researchers are given reasonable time to report vulnerabilities
2. We provide timely acknowledgment and assessment
3. Fixes are developed and tested privately
4. Public disclosure is coordinated with reporter
5. Credit is given to reporter (unless anonymity requested)

### Public Disclosure

After a fix is released:
- Security advisory is published on GitHub
- CVE is requested for High/Critical vulnerabilities
- Release notes clearly indicate security fixes
- Documentation is updated with mitigation guidance

## Security Hall of Fame

We gratefully acknowledge security researchers who have responsibly disclosed vulnerabilities:

<!-- List will be populated as reports are received and fixed -->

*No vulnerabilities have been reported yet.*

## Contact

For security-related questions or concerns:
- Review this security policy
- Check [GitHub Security Advisories](https://github.com/dantte-lp/ocserv-agent/security/advisories)
- See [Contributing Guidelines](.github/CONTRIBUTING.md) for general contributions

## Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [Go Security Best Practices](https://golang.org/doc/security/)
- [gRPC Security Guide](https://grpc.io/docs/guides/auth/)

---

**Last Updated**: 2025-10-23
**Security Policy Version**: 1.0
