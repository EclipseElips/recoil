# Changelog

All notable changes to recoil are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and recoil aims to
adhere to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.2] - 2026-06-29

### Changed
- README now shows how to run a prebuilt download per platform (Windows PATH,
  macOS quarantine clear, Linux `chmod +x`).

## [1.0.1] - 2026-06-29

### Changed
- Rewrote the bundled skill so agents reliably reach for it: a symptom-keyed
  trigger description, a no-hesitation "when to fire" rule, worked recall/encode
  examples, and an explicit "act on what fires" step.

## [1.0.0] - 2026-06-29

First release.

### Added
- `encode` / `recall` over a plain-text TSV store, matched by deterministic
  keyword cue-overlap — no embeddings, no network.
- `guard` warns before a known-bad change (errors, reverts, corrections, not
  plain notes), with a configurable cue-overlap floor (`--min-overlap`, default 2).
- `watch -- <cmd>` records a command as a flinch when it exits non-zero.
- `decay` forgets memories whose strength has faded; recall renews the useful ones.
- `hook --install` wires git pre-commit (guard) and post-commit (revert capture)
  hooks without overwriting existing ones.
- `init`, `list`, and `version` commands.
- Packaged as a Claude Code plugin and a Codex plugin, with a bundled skill and a
  warn-only pre-edit guard hook. `AGENTS.md` fallback for other agents.

[Unreleased]: https://github.com/EclipseElips/recoil/compare/v1.0.2...HEAD
[1.0.2]: https://github.com/EclipseElips/recoil/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/EclipseElips/recoil/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/EclipseElips/recoil/releases/tag/v1.0.0
