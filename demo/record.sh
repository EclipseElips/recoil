#!/usr/bin/env bash
# Regenerate the README demo GIFs in assets/. Every recoil command below really
# runs; the `#` lines are plain annotations, nothing is staged. Recording uses
# asciinema + agg (static binaries, fetched into demo/.tools) — no browser, no
# ffmpeg, no sudo. Needs: go, curl, and a monospace font. Linux and macOS.
#   ./demo/record.sh
set -euo pipefail
cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"   # repo root
TOOLS="$PWD/demo/.tools"; mkdir -p "$TOOLS"; export PATH="$TOOLS:$PATH"
ASCIINEMA_VER=v3.2.1; AGG_VER=v1.9.0

go build -o "$TOOLS/recoil" .

case "$(uname -s)" in Linux) sys=unknown-linux-musl;; Darwin) sys=apple-darwin;; *) echo "unsupported OS"; exit 1;; esac
case "$(uname -m)" in x86_64|amd64) arch=x86_64;; arm64|aarch64) arch=aarch64;; *) echo "unsupported arch"; exit 1;; esac
dl(){ [ -x "$2" ] || { echo "fetching $(basename "$2")"; curl -fsSL "$1" -o "$2"; chmod +x "$2"; }; }
dl "https://github.com/asciinema/asciinema/releases/download/$ASCIINEMA_VER/asciinema-$arch-$sys" "$TOOLS/asciinema"
dl "https://github.com/asciinema/agg/releases/download/$AGG_VER/agg-$arch-$sys" "$TOOLS/agg"

D=$(mktemp -d); mkdir -p assets

# ---------- basics: encode a lesson, recall it, guard a known-bad change ----------
cat > "$D/cli.sh" <<'EOS'
#!/usr/bin/env bash
cd "$(mktemp -d)"
P=$'\033[32m$\033[0m'
sleep 0.6
printf '%s recoil init\n' "$P"; recoil init; sleep 1.0
printf '%s recoil encode --trigger test-fail \\\n' "$P"
printf '    --gist "Don'\''t name a Unity folder Build/, .gitignore untracks it" \\\n'
printf '    --cue  "unity build folder gitignore"\n'
recoil encode --trigger test-fail --gist "Don't name a Unity folder Build/, .gitignore untracks it" --cue "unity build folder gitignore"; sleep 1.4
printf '%s echo "editing .gitignore and a new Build dir" | recoil recall\n' "$P"
echo "editing .gitignore and a new Build dir" | recoil recall; sleep 2.4
printf '%s recoil guard --files Build/Player.cs,.gitignore\n' "$P"
recoil guard --files Build/Player.cs,.gitignore; sleep 3.0
EOS

# ---------- the loop an agent runs: guard before editing, encode on a correction, recall next time ----------
cat > "$D/agent.sh" <<'EOS'
#!/usr/bin/env bash
cd "$(mktemp -d)"
P=$'\033[32m$\033[0m'; C=$'\033[90m'; R=$'\033[0m'
recoil init >/dev/null 2>&1
recoil encode --trigger test-fail --gist "Don't name a Unity folder Build/, .gitignore untracks it" --cue "unity build folder gitignore" >/dev/null 2>&1
sleep 0.6
printf '%s# before it touches files, the agent guards -- and catches a past mistake%s\n' "$C" "$R"; sleep 0.9
printf '%s recoil guard --files Build/Player.cs,.gitignore\n' "$P"
recoil guard --files Build/Player.cs,.gitignore; sleep 2.4
printf '%s# the user corrects it on something new -- it records the lesson%s\n' "$C" "$R"; sleep 0.9
printf '%s recoil encode --trigger correction \\\n' "$P"
printf '    --gist "Run EditMode tests from the CLI, not the GUI runner" \\\n'
printf '    --cue  "unity test editmode runner cli"\n'
recoil encode --trigger correction --gist "Run EditMode tests from the CLI, not the GUI runner" --cue "unity test editmode runner cli"; sleep 1.6
printf '%s# a later task looks familiar -- it recalls before starting%s\n' "$C" "$R"; sleep 0.9
printf '%s recoil recall --situation "add an EditMode test for the runner"\n' "$P"
recoil recall --situation "add an EditMode test for the runner"; sleep 3.0
EOS

# ---------- decay: unused lessons fade and get forgotten ----------
cat > "$D/decay.sh" <<'EOS'
#!/usr/bin/env bash
cd "$(mktemp -d)"; mkdir -p .recoil
P=$'\033[32m$\033[0m'; C=$'\033[90m'; R=$'\033[0m'
now=$(date +%s); old=$((now-200*86400)); recent=$((now-4*86400))
{ printf 'r1\t%s\tcorrection\t3\t2\t%s\tasync deadlock editmode throwsasync\tAssert.ThrowsAsync hangs the EditMode runner - use try/catch\n' "$old" "$recent"
  printf 'r2\t%s\tmanual\t1\t0\t%s\tformatting tabs spaces nit\tprefer tabs over spaces in this repo\n' "$old" "$old"; } > .recoil/store.tsv
sleep 0.6
printf '%s# unused lessons lose strength -- recoil list shows the str= for each%s\n' "$C" "$R"; sleep 0.9
printf '%s recoil list\n' "$P"; recoil list; sleep 2.8
printf '%s# forget the ones that have faded below the floor%s\n' "$C" "$R"; sleep 0.9
printf '%s recoil decay --dry-run\n' "$P"; recoil decay --dry-run; sleep 2.4
printf '%s recoil decay\n' "$P"; recoil decay; sleep 2.8
EOS

render(){ # label  script  gif  rows
  asciinema rec --overwrite -c "bash $2" "$D/$1.cast"
  agg --cols 100 --rows "$4" --font-size 16 --font-family "DejaVu Sans Mono" "$D/$1.cast" "$3"
  echo "wrote $3"
}
render cli   "$D/cli.sh"   assets/demo.gif        18
render agent "$D/agent.sh" assets/demo-agent.gif  18
render decay "$D/decay.sh" assets/demo-decay.gif  18
