# recoil

Memory for AI coding agents. It remembers the things that go wrong — a failed
command, a revert, a correction — and reminds you when you're about to hit them
again. One Go binary, a plain text file, no embeddings.

## What it does

- Remembers a lesson tied to the situation it happened in — the files, the error
  text, the keywords around it.
- Brings that lesson back when you're in a similar situation again, matched by
  plain keyword overlap (an unrelated task gets nothing).
- Records failures for you: `recoil watch -- <cmd>` remembers anything that exits
  non-zero, no manual step.
- Records git reverts for you, via a post-commit hook.
- Surfaces the lessons that keep mattering — each recall makes one a little louder.
- Lets unused lessons fade, and `recoil decay` clears out the ones that stopped
  mattering — recall keeps the useful ones alive.
- Keeps everything in one plain-text file you can read and edit by hand.

## Build

```sh
go build -o recoil .
```

Stdlib only, so that's the whole build.

## Use

Record a lesson with the situation it happened in:

```sh
recoil encode --trigger test-fail \
  --gist "Don't name a Unity folder Build/, .gitignore untracks it" \
  --cue  "unity build folder gitignore"
```

Later, recall by what you're doing now. Matching is plain keyword overlap, so an
unrelated task gets nothing back:

```
$ echo "editing .gitignore and a new Build dir" | recoil recall
>> Don't name a Unity folder Build/, .gitignore untracks it
   [test-fail w=2 hits=0] matched: build gitignore unity
```

A lesson gets a little louder each time it's recalled, so the ones that keep
mattering stay near the top.

## Auto-capture

Wrap a command. If it fails, recoil records it for you:

```sh
recoil watch -- go test ./...
```

Install a git hook to record reverts:

```sh
recoil hook --install
```

## Forgetting

A lesson's strength fades the longer it goes unused — it halves every 30 unused
days by default. Recalling a lesson resets that clock and bumps its count, so the
ones you keep hitting stay strong while the rest fade. Surprise weight and re-fire
count both raise strength, so a hard-won correction outlives a routine note. Clear
out the faded ones:

```sh
recoil decay --dry-run    # show what would go
recoil decay              # forget it
```

## Commands

```
recoil init                       create the store
recoil encode --gist .. --cue ..  record a lesson
recoil recall [--situation ..]    show matching lessons (also reads stdin)
recoil decay [--dry-run]          forget lessons that have faded
recoil watch -- <cmd>             run a command, record it if it fails
recoil hook [--install]           git post-commit hook for reverts
recoil list                       show everything stored
```

Triggers and their default weights: `correction` 3, `revert` 2.5, `test-fail` 2,
`error` 1.5, `manual` 1. Higher weight means it surfaces sooner.

## Store

Plain TSV at `$RECOIL_DIR/store.tsv` (default `./.recoil/store.tsv`), one line
per lesson: `id  created  trigger  weight  hits  last  cue  gist`. You can read
it and edit it.

## License

MIT.
