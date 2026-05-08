# Security Policy

## Supported versions

ProvenArch is currently an MVP beta / pre-release foundation. Security fixes are supported for the latest `v0.1.x` release line and the `main` branch.

| Version | Supported |
|---|---|
| `v0.1.x` | Yes |
| older releases | No |

## Reporting a vulnerability

Do not open a public GitHub issue for a suspected vulnerability.

Use GitHub private vulnerability reporting from the repository Security tab:

https://github.com/GrinRus/ProvenArch/security/advisories/new

If private vulnerability reporting is unavailable, contact a maintainer through a private channel listed on the repository owner profile and include only non-sensitive summary details until a private channel is established.

## Expected response

- Acknowledgement target: 3 business days.
- Initial triage target: 7 business days.
- Fix target: depends on severity and exploitability.
- Disclosure: coordinated disclosure after a fix is available, unless active exploitation requires earlier user notice.

## Scope

In scope:
- `acp` CLI and local server behavior.
- GitHub Releases artifacts and `install.sh`.
- Workspace file handling, local repo resolution, redaction, and runtime write boundaries.
- Embedded React UI shipped inside the binary.

Out of scope for the MVP:
- Hosted/SaaS control plane security.
- Security/compliance enforcement claims.
- Third-party headless provider vulnerabilities outside ACP invocation, redaction, and artifact handling.

## Safe harbor

Good-faith research that avoids privacy violations, data destruction, service disruption, and persistence is welcome. Stop testing and report privately if you encounter sensitive data, credentials, or a path to destructive behavior.
