# AGENTS.md

Guidance for AI coding agents working in this repo. (Codex reads this file
automatically; other agents can be pointed at it.)

## What recoil is

A local-first operational memory CLI for coding agents: one Go binary, one
plain-text store (`.recoil/store.tsv`), no embeddings, no network.

## Use recoil on yourself

- Before editing or running tests: `recoil recall --files <files>` and
  `recoil guard --files <files>`; read the output before you act.
- After a correction, an unexpected test/build failure, or a revert:
  `recoil encode --trigger <correction|test-fail|revert> --gist "<lesson>" --cue "<files,errors,systems>"`.

Keep the gist one line. Make the cue the tokens the next occurrence will share.
Only encode when the loop was actually surprised.

## Keep recoil small (contributing constraints)

- Stdlib only. `go.mod` has no `require` block — keep it that way.
- No embeddings, no network, no model calls. Recall is deterministic keyword
  overlap. That's the whole point.
- One binary, one human-readable store.
- Before a PR: `go build ./...`, `go vet ./...`, and `go test ./...` all pass.
  New behavior has a test — the scoring and parsing logic is pure and easy to
  test (see `main_test.go`).
