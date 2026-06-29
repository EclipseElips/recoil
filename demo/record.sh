#!/usr/bin/env bash
# Regenerate the README demo GIFs in assets/ as REAL recorded sessions:
# demo/typer.py drives a genuine interactive bash and types each command, so the
# terminal echoes it and recoil really runs -- nothing is staged. asciinema
# records the session and agg renders the GIF. No browser, no ffmpeg, no sudo.
# Needs: go, python3, curl, and a monospace font. Linux and macOS.
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
one(){ # demo  gif
  asciinema rec --overwrite -c "python3 demo/typer.py $1" "$D/$1.cast"
  agg --cols 100 --rows 24 --font-size 16 --font-family "DejaVu Sans Mono" "$D/$1.cast" "$2"
  echo "wrote $2"
}
one cli   assets/demo.gif
one agent assets/demo-agent.gif
one decay assets/demo-decay.gif
