# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please DO NOT report security vulnerabilities through public GitHub issues.**

We take security seriously and appreciate your efforts to responsibly disclose your findings.

### How to Report

Please report security vulnerabilities by email to:

**📧 jvreagan@gmail.com**

### What to Include

To help us triage and fix the issue quickly, please include:

- **Description** of the vulnerability
- **Steps to reproduce** the issue
- **Potential impact** (what an attacker could do)
- **Suggested fix** (if you have one)
- **Your contact information** (for follow-up questions)

### What to Expect

- **Response Time:** You will receive an acknowledgment within **48 hours**
- **Updates:** We'll keep you informed of our progress
- **Disclosure:** We'll coordinate with you on public disclosure timing
- **Credit:** We'll acknowledge your contribution (unless you prefer to remain anonymous)

### Security Update Process

1. **Triage:** We verify and assess the vulnerability
2. **Fix:** We develop and test a patch
3. **Release:** We release a security update
4. **Disclosure:** We publish a security advisory
5. **Credit:** We thank you in the release notes

## Security Best Practices

When using cloud-deploy, we recommend:

### Credential Management

- **Never commit credentials** to version control
- **Use environment variables** for all secrets
- **Enable Vault integration** for production deployments
- **Rotate API keys** regularly (at least quarterly)

### GitHub Actions / CI/CD

- **Use GitHub Secrets** for cloud credentials
- **Never log secrets** in CI/CD output
- **Use least-privilege credentials** (minimal permissions needed)

### Deployment Security

- **Review generated manifests** before deploying
- **Use separate credentials** for dev/staging/production
- **Enable audit logging** in your cloud provider
- **Monitor deployment activity** for anomalies

### Example: Secure Manifest

```yaml
version: "1.0"

# ✅ GOOD: Use environment variables
provider:
  credentials:
    source: environment  # Reads AWS_ACCESS_KEY_ID, etc.

# ✅ GOOD: Use Vault for secrets
secrets:
  - name: DATABASE_URL
    vault_path: secret/data/myapp/database
    vault_key: url

# ❌ BAD: Never hardcode credentials
# provider:
#   credentials:
#     access_key_id: AKIAIOSFODNN7EXAMPLE  # DON'T DO THIS!
```

## Known Security Considerations

### Multi-Cloud Credentials

cloud-deploy requires cloud provider credentials to deploy resources. We recommend:

- **Development:** Use time-limited credentials or AWS SSO
- **Production:** Use Vault integration or managed identities
- **CI/CD:** Use OIDC federation where available (GitHub Actions, etc.)

### Container Registry Access

cloud-deploy pushes Docker images to cloud registries using API calls (no CLI dependencies). Ensure:

- Container registries use private access
- Use separate registries for dev/prod environments
- Enable vulnerability scanning on registries

### Manifest Files

Deployment manifests may contain sensitive configuration. Best practices:

- Add `deploy-manifest.yaml` to `.gitignore` (already included)
- Use example files with placeholder values for version control
- Store production manifests in secure locations (Vault, AWS Secrets Manager)

## Security Features

### Built-in Protections

- ✅ **No hardcoded credentials** in codebase
- ✅ **Environment variable support** for all providers
- ✅ **HashiCorp Vault integration** for secret management
- ✅ **Dependabot enabled** for dependency updates
- ✅ **Secret scanning enabled** on GitHub
- ✅ **Enhanced .gitignore** prevents credential commits

### Regular Updates

We maintain this project with:

- Weekly Dependabot updates for dependencies
- Prompt security patches when vulnerabilities are found
- Regular security audits of the codebase

## Severity Classification

We use the CVSS 3.1 scoring system:

| Severity | CVSS Score | Response Time |
|----------|------------|---------------|
| Critical | 9.0-10.0   | 24 hours      |
| High     | 7.0-8.9    | 7 days        |
| Medium   | 4.0-6.9    | 30 days       |
| Low      | 0.1-3.9    | Best effort   |

## Security Advisories

Published security advisories can be found at:
https://github.com/jvreagan/cloud-deploy/security/advisories

## Questions?

If you have questions about this security policy, please contact jvreagan@gmail.com

---

**Thank you for helping keep cloud-deploy and our users secure!** 🔒
