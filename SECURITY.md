# Security Policy

## Supported versions

chip is pre-1.0. Only the latest tagged release is supported; please upgrade to
the most recent release before reporting an issue.

## Reporting a vulnerability

Please report security vulnerabilities **privately** — do not open a public
issue for an undisclosed vulnerability.

- **Preferred:** open a private report via GitHub Security Advisories — the
  **Security** tab on the repository, then **Report a vulnerability**.
- **Or** email **jack@spirou.io** with details and steps to reproduce.

Expect an initial acknowledgement within a few days. Once a fix is ready a new
release is cut and the advisory is published, with credit to the reporter unless
anonymity is requested.

## Verifying releases

Every release publishes `checksums.txt` (SHA-256 over each archive and binary), a
software bill of materials (SBOM), and a keyless Sigstore build-provenance
attestation. Verify a downloaded artifact against the attestation with:

```sh
gh attestation verify <file> --repo jackspirou/chip
```
