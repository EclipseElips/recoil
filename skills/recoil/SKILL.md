---
name: recoil
description: >-
  Operational memory for this repo via the `recoil` CLI. Use it BEFORE editing
  code, running tests, or starting a task: run `recoil recall` and
  `recoil guard <files>` to surface past mistakes for the files in scope. Use it
  AFTER a user correction, an unexpected build/test failure, or a revert: run
  `recoil encode` to record the lesson. Do NOT use it for read-only questions
  that change nothing.
---

# recoil — operational memory for coding agents

`recoil` is a local CLI backed by a plain-text store (`.recoil/store.tsv`). It
remembers what went wrong here before — a failed command, a revert, a correction
— and surfaces it when you're about to hit the same thing again. Matching is
deterministic keyword overlap: no embeddings, no network.

## Make sure it's installed

If `recoil` isn't on your PATH:

```sh
go install github.com/EclipseElips/recoil@latest
```

Once per repo:

```sh
recoil init
```

## Before you change code or run tests

Surface anything you've been burned on for the files in play, and read it before
editing:

```sh
recoil recall --files path/one.go,path/two.go
recoil guard  --files path/one.go,path/two.go
```

`guard` only warns about things that actually went wrong (errors, reverts,
corrections) — not plain notes.

## After something goes wrong — record the lesson

Keep the gist one actionable line. For the cue, use the tokens the next
occurrence will share: file names, error text, system names.

- User corrected you on something non-obvious:

  ```sh
  recoil encode --trigger correction --gist "<lesson>" --cue "<files errors systems>"
  ```

- A build or test failed unexpectedly:

  ```sh
  recoil encode --trigger test-fail --gist "<lesson>" --cue "<...>"
  ```

  or wrap the command so it records itself on failure:

  ```sh
  recoil watch -- go test ./...
  ```

- A change got reverted:

  ```sh
  recoil encode --trigger revert --gist "<lesson>" --cue "<...>"
  ```

## Keep it useful

- `recoil list` — show everything stored.
- `recoil decay` — forget lessons that have faded (unused ones lose strength).
- `recoil hook --install` — wire the git pre/post-commit guard in this repo.

Only encode when the loop was actually surprised. Routine notes don't belong
here — they belong in your normal docs.
