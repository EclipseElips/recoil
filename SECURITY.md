# Security Policy

recoil runs entirely on your machine — no network, no telemetry, no model calls
— and its store is a plain-text file you can read. The meaningful surface is that
it installs git hooks (`recoil hook --install`) and runs the command you hand it
(`recoil watch -- <cmd>`).

## Reporting a vulnerability

Please report security issues privately through GitHub's
[Private Vulnerability Reporting](https://github.com/EclipseElips/recoil/security/advisories/new)
rather than opening a public issue. You can expect an acknowledgement within a
few days and an update as a fix lands.
