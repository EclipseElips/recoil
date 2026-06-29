#!/usr/bin/env python3
# Drive a REAL interactive bash in a pty and genuinely type the demo commands,
# so the terminal echoes them itself and recoil really runs. Nothing is faked:
# the prompt, the typed command, and the output are all the real session.
# Run under asciinema:  asciinema rec -c "python3 demo/typer.py <cli|agent|decay>" out.cast
import os, pty, select, time, sys, fcntl, termios, struct, tempfile

now = int(time.time()); old = now - 200 * 86400; recent = now - 4 * 86400

# Stores seeded (off camera) before recording, so the demo starts mid-history.
SEED = {
    "agent": [
        "r0\t%d\ttest-fail\t2\t0\t%d\tunity build folder gitignore\t"
        "Don't name a Unity folder Build/, .gitignore untracks it\n" % (now, now),
    ],
    "decay": [
        "r1\t%d\tcorrection\t3\t2\t%d\tasync deadlock editmode throwsasync\t"
        "Assert.ThrowsAsync hangs the EditMode runner - use try/catch\n" % (old, recent),
        "r2\t%d\tmanual\t1\t0\t%d\tformatting tabs spaces nit\t"
        "prefer tabs over spaces in this repo\n" % (old, old),
    ],
}

# (keystrokes, pause_after_seconds). A trailing "\\" types a real line continuation.
LINES = {
    "cli": [
        ("recoil init", 1.0),
        ("recoil encode --trigger test-fail \\", 0.25),
        ("  --gist \"Don't name a Unity folder Build/, .gitignore untracks it\" \\", 0.25),
        ("  --cue  \"unity build folder gitignore\"", 1.4),
        ("echo \"editing .gitignore and a new Build dir\" | recoil recall", 2.4),
        ("recoil guard --files Build/Player.cs,.gitignore", 3.0),
    ],
    "agent": [
        ("# before it touches files, the agent guards -- and catches a past mistake", 0.9),
        ("recoil guard --files Build/Player.cs,.gitignore", 2.4),
        ("# the user corrects it on something new -- it records the lesson", 0.9),
        ("recoil encode --trigger correction \\", 0.25),
        ("  --gist \"Run EditMode tests from the CLI, not the GUI runner\" \\", 0.25),
        ("  --cue  \"unity test editmode runner cli\"", 1.6),
        ("# a later task looks familiar -- it recalls before starting", 0.9),
        ("recoil recall --situation \"add an EditMode test for the runner\"", 3.0),
    ],
    "decay": [
        ("# unused lessons lose strength -- recoil list shows the str= for each", 0.9),
        ("recoil list", 2.8),
        ("# forget the ones that have faded below the floor", 0.9),
        ("recoil decay --dry-run", 2.4),
        ("recoil decay", 2.8),
    ],
}

demo = sys.argv[1]
work = tempfile.mkdtemp()
if demo in SEED:
    os.makedirs(os.path.join(work, ".recoil"), exist_ok=True)
    with open(os.path.join(work, ".recoil", "store.tsv"), "w") as f:
        f.write("".join(SEED[demo]))
rc = os.path.join(work, ".democ")
with open(rc, "w") as f:
    f.write("PS1='\\[\\033[32m\\]$\\[\\033[0m\\] '\nPS2='> '\n")
os.chdir(work)

pid, master = pty.fork()
if pid == 0:
    os.environ["TERM"] = "xterm-256color"
    os.execvp("bash", ["bash", "--noprofile", "--rcfile", rc, "-i"])
    os._exit(1)

fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 100, 0, 0))


def pump(duration):
    end = time.time() + duration
    while True:
        to = end - time.time()
        if to <= 0:
            return True
        try:
            r, _, _ = select.select([master], [], [], to)
        except (OSError, ValueError):
            return False
        if master in r:
            try:
                data = os.read(master, 65536)
            except OSError:
                return False
            if not data:
                return False
            os.write(1, data)


pump(0.7)
for text, delay in LINES[demo]:
    for ch in text:
        os.write(master, ch.encode())
        pump(0.02)
    os.write(master, b"\r")
    pump(delay)
os.write(master, b"exit\r")
t0 = time.time()
while time.time() - t0 < 3 and pump(0.3):
    pass
try:
    os.close(master)
except OSError:
    pass
try:
    os.waitpid(pid, 0)
except OSError:
    pass
