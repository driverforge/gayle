# Security Policy

## Reporting a vulnerability

Please report security issues **privately** — do not open a public issue.

- **Preferred:** GitHub's [private vulnerability reporting](https://docs.github.com/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) on this repository (**Security → Report a vulnerability**).
- **Or** email **oss@driverforge.com**.

Please include a description, reproduction steps, and the affected command or code path where you can. We'll acknowledge your report and keep you posted on the fix.

## Scope

This policy covers the **gayle CLI** in this repository. gayle reads and writes configuration and secrets in AWS SSM Parameter Store and Azure Key Vault using the **operator's own ambient credentials** (the standard AWS credential chain / Azure `DefaultAzureCredential`); it ships no credentials of its own and talks to no Driverforge services. Of particular interest: anything that could leak a secret value (to output, logs, or files beyond what the operator asked for), write to or delete parameters outside the configured paths, or misreport success (exit 0) for an operation that did not verifiably complete.

Vulnerabilities in the AWS or Azure SDKs themselves are best reported upstream — but if gayle's use of them makes a flaw exploitable, we want to know too.

Note that some outputs are sensitive **by design** and handled by the operator: `gayle export` writes secret values to a file, and `gayle fetch` prints them to stdout. Reports about those behaving as documented are not vulnerabilities.

## Supported versions

| Version | Supported |
| ------- | --------- |
| 6.x (Go) | ✅ current — fixes land here |
| < 6 (Node, `@driverforge/gayle` on npm) | ❌ retired and deprecated |

Please confirm the issue against the latest release or current `main` before reporting.
