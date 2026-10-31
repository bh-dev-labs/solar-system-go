# Security policy

## Reporting a vulnerability

Please report vulnerabilities privately through GitHub's
**Security → Report a vulnerability** form on this repository
(private vulnerability reporting). Do not open a public issue.

Include steps to reproduce and the affected version or commit.

## Supported versions

Only the latest commit on `main` is supported.

## Scope notes

The server binds to `127.0.0.1:8080` by default and serves static files plus
read-only JSON computed in-process. It uses only the Go standard library.
Binding it to a public interface with `-addr` is at your own risk.
