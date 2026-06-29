#!/bin/sh
# recoil PreToolUse guard: warn (never block) before editing something recoil has
# been burned on before. Claude Code pipes the tool-call JSON on stdin; recoil
# tokenizes it and warns only on a real overlap with a past failure.
#
# This script is deliberately defensive: if recoil isn't installed, or it has
# nothing to say, it exits 0 and stays out of the way. It never blocks an edit.
command -v recoil >/dev/null 2>&1 || exit 0
recoil guard 2>&1 || true
exit 0
