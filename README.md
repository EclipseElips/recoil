# recoil

[![ci](https://github.com/EclipseElips/recoil/actions/workflows/ci.yml/badge.svg)](https://github.com/EclipseElips/recoil/actions/workflows/ci.yml)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
![Claude Code plugin](https://img.shields.io/badge/Claude%20Code-plugin-d97757)
![Codex skill](https://img.shields.io/badge/Codex-skill-412991)

Memory for AI coding agents. It remembers the things that go wrong — a failed
command, a revert, a correction — and reminds your agent when it's about to hit
them again. One Go binary, a plain text file, no embeddings.

![recoil: encode a lesson, recall it, then guard a known-bad change](assets/demo.gif)

## What it does

- Remembers a lesson tied to the situation it happened in — the files, the error
  text, the keywords around it.
- Brings that lesson back when your agent is in a similar situation again, matched
  by plain keyword overlap (an unrelated task gets nothing).
- Records failures automatically: `recoil watch -- <cmd>` remembers anything that
  exits non-zero, no manual step.
- Records git reverts automatically, via a post-commit hook.
- Warns the agent before it repeats a known-bad change — a git pre-commit hook
  that flags when a change touches something that went wrong here before.
- Surfaces the lessons that keep mattering — each recall makes one a little louder.
- Lets unused lessons fade, and `recoil decay` clears out the ones that stopped
  mattering — recall keeps the useful ones alive.
- Keeps everything in one plain-text file you can read and edit by hand.

## Install

recoil is a single binary that needs to be on your `PATH`. No Go toolchain needed
for the prebuilt builds:

```sh
curl -sSfL https://raw.githubusercontent.com/EclipseElips/recoil/main/install.sh | sh -s -- -b /usr/local/bin
```

Or download an archive for your platform from the
[releases page](https://github.com/EclipseElips/recoil/releases).

recoil is a **command-line tool** — you run it from a terminal. After you unpack
a prebuilt archive, put the binary somewhere on your `PATH` and run it from a
shell:

**Windows.** 

```powershell
.\recoil.exe init
.\recoil.exe version
```

Move `recoil.exe` into a folder on your `PATH` to run `recoil` from anywhere.

**macOS.**

```sh
xattr -d com.apple.quarantine ./recoil
chmod +x ./recoil
./recoil version
```

**Linux.**

```sh
chmod +x ./recoil
./recoil version
```

With a Go toolchain:

```sh
go install github.com/EclipseElips/recoil@latest
```

From source — stdlib only, so that's the whole build:

```sh
go build -o recoil .
```

Then create the store, once per repo:

```sh
recoil init
```

## Use it with your coding agent

recoil ships as a skill, so the agent knows when to reach for it: recall and
guard before it touches your files, encode a lesson when something goes wrong.

![the loop an agent runs: guard before it edits, encode on a correction, recall on the next related task](assets/demo-agent.gif)

**Claude Code** — recoil is submitted to Anthropic's community plugin directory.
Once it's listed there:

```
/plugin marketplace add anthropics/claude-plugins-community
/plugin install recoil@claude-community
```

Until then — or to install straight from this repo — add it as its own
marketplace:

```
/plugin marketplace add EclipseElips/recoil
/plugin install recoil@recoil
```

Either way you get the skill plus a warn-only pre-edit guard hook. The recoil
binary itself still needs to be on your `PATH` (see Install). Update later from
the **Marketplaces** tab in `/plugin`.

**Codex** — install the plugin from this repo as a marketplace source, or just
drop the skill into any repo:

```sh
cp -r skills/recoil .agents/skills/recoil
```

A short `AGENTS.md` stanza is included as a fallback for older Codex builds.

**Any agent** — the skill is just instructions wrapped around the CLI. Point your
agent at `recoil recall`, `recoil guard`, and `recoil encode` (see Commands).

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
   [test-fail w=2 hits=0] matched: build gitignore
```

A lesson gets a little louder each time it's recalled, so the ones that keep
mattering stay near the top.

## Auto-capture

Wrap a command. If it fails, recoil records it:

```sh
recoil watch -- go test ./...
```

## Warn before repeating it

`recoil guard` checks what's about to change against the things that went wrong
before — errors, reverts, corrections, not plain notes — and warns if a change is
walking back into one:

```sh
recoil guard --files src/foo.go
# recoil: been burned here before — Assert.ThrowsAsync hangs the EditMode runner
```

A warning needs at least two overlapping cue tokens by default (`--min-overlap`),
so one coincidental shared word won't trip it.

With no arguments in a git repo it checks the staged files, so it runs as a
pre-commit hook. Wire up both hooks — the pre-commit guard and the post-commit
revert recorder — with one command (it won't overwrite hooks you already have):

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

![unused lessons fade; `recoil decay` forgets the ones that drop below the floor](assets/demo-decay.gif)

## Commands

```
recoil init                       create the store
recoil encode --gist .. --cue ..  record a lesson
recoil recall [--situation ..]    show matching lessons (also reads stdin)
recoil guard [--files a,b]        warn if a change repeats a known-bad one
recoil decay [--dry-run]          forget lessons that have faded
recoil watch -- <cmd>             run a command, record it if it fails
recoil hook [--install]           git pre/post-commit hooks (warn, record reverts)
recoil list                       show everything stored
```

Triggers and their default weights: `correction` 3, `revert` 2.5, `test-fail` 2,
`error` 1.5, `manual` 1. Higher weight means it surfaces sooner.

## Store

Plain TSV at `$RECOIL_DIR/store.tsv` (default `./.recoil/store.tsv`), one line
per lesson: `id  created  trigger  weight  hits  last  cue  gist`. You can read
it and edit it.

## License

MIT. Issues and PRs welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). To report a
security issue, see [SECURITY.md](SECURITY.md).
