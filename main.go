// Command recoil is a local-first memory for AI coding agents.
//
// recoil remembers the things that go wrong in the development loop — a failed
// command, a revert, a correction — as a cue (the situation it happened in) plus
// a gist (the lesson). It surfaces those gists when the current situation looks
// the same, and warns before you repeat a known-bad change. Matching is
// deterministic keyword cue-overlap: no embeddings, no model, no network.
// Memories fade when they stop being useful. One static binary, one plain-text
// store.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// version is the recoil release. Release builds stamp the git tag over it via
// -ldflags "-X main.version=..."; plain `go build`/`go install` keep this value.
var version = "1.0.1"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		cmdInit(os.Args[2:])
	case "encode":
		cmdEncode(os.Args[2:])
	case "recall":
		cmdRecall(os.Args[2:])
	case "decay":
		cmdDecay(os.Args[2:])
	case "guard":
		cmdGuard(os.Args[2:])
	case "watch":
		cmdWatch(os.Args[2:])
	case "hook":
		cmdHook(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println("recoil", version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `recoil `+version+` — local-first memory for AI coding agents

usage:
  recoil init
  recoil encode --gist "<lesson>" --cue "<tokens>" [--trigger T] [--weight N]
  recoil recall [--situation "<text>"] [--files a,b] [--top N]   (also reads stdin)
  recoil decay [--floor F] [--half-life D] [--dry-run]    forget faded memories
  recoil guard [--files a,b] [--situation "<text>"] [--min-overlap N] [--block]    warn before a known-bad change
  recoil watch -- <command> [args...]    run a command; remember it if it fails
  recoil hook [--install]                git pre/post-commit hooks (warn, record reverts)
  recoil list
  recoil version

triggers (default weight): correction=3  revert=2.5  test-fail=2  error=1.5  manual=1
store: $RECOIL_DIR/store.tsv  (default ./.recoil/store.tsv)
`)
}

// --- store location ---

func storeDir() string {
	if d := os.Getenv("RECOIL_DIR"); d != "" {
		return d
	}
	return ".recoil"
}

func storePath() string { return filepath.Join(storeDir(), "store.tsv") }

// --- record model ---

// record is one remembered flinch: a cue (the situation that triggered it) plus
// a gist (the lesson), with salience bookkeeping for ranking and decay.
type record struct {
	ID      string
	Created int64
	Trigger string
	Weight  float64
	Hits    int
	Last    int64
	Cue     string // space-separated, lowercased tokens
	Gist    string
}

func clean(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func (r record) tsv() string {
	return strings.Join([]string{
		r.ID,
		strconv.FormatInt(r.Created, 10),
		r.Trigger,
		strconv.FormatFloat(r.Weight, 'g', -1, 64),
		strconv.Itoa(r.Hits),
		strconv.FormatInt(r.Last, 10),
		clean(r.Cue),
		clean(r.Gist),
	}, "\t")
}

// parseRecord parses one TSV line. It returns false on a wrong column count OR a
// non-numeric numeric field, so a hand-edited store with a damaged line is
// skipped rather than silently loaded as zero values.
func parseRecord(line string) (record, bool) {
	f := strings.Split(line, "\t")
	if len(f) != 8 {
		return record{}, false
	}
	created, err1 := strconv.ParseInt(f[1], 10, 64)
	weight, err2 := strconv.ParseFloat(f[3], 64)
	hits, err3 := strconv.Atoi(f[4])
	last, err4 := strconv.ParseInt(f[5], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return record{}, false
	}
	return record{
		ID: f[0], Created: created, Trigger: f[2], Weight: weight,
		Hits: hits, Last: last, Cue: f[6], Gist: f[7],
	}, true
}

func loadRecords() ([]record, error) {
	f, err := os.Open(storePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var recs []record
	skipped := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if r, ok := parseRecord(line); ok {
			recs = append(recs, r)
		} else {
			skipped++
		}
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "recoil: warning: skipped %d malformed line(s) in %s\n", skipped, storePath())
	}
	return recs, sc.Err()
}

func saveRecords(recs []record) error {
	if err := os.MkdirAll(storeDir(), 0o755); err != nil {
		return err
	}
	tmp := storePath() + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, r := range recs {
		fmt.Fprintln(w, r.tsv())
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, storePath())
}

func appendRecord(r record) error {
	if err := os.MkdirAll(storeDir(), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(storePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, r.tsv())
	return err
}

// --- tokenization: deterministic, no embeddings ---

var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "to": true,
	"of": true, "in": true, "on": true, "for": true, "is": true, "it": true,
	"this": true, "that": true, "with": true, "was": true, "at": true, "by": true,
	"i": true, "im": true, "my": true, "we": true, "you": true, "should": true,
}

// tokenize lowercases and splits on any non-alphanumeric run, dropping stopwords
// and single characters. "build-index.mjs" -> {build, index, mjs}.
func tokenize(s string) map[string]bool {
	set := map[string]bool{}
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		t := strings.ToLower(cur.String())
		cur.Reset()
		if len(t) < 2 || stopwords[t] {
			return
		}
		set[t] = true
	}
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			cur.WriteRune(c)
		} else {
			flush()
		}
	}
	flush()
	return set
}

func sortedTokens(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// --- strength, scoring, decay, guard (pure: no I/O, so they are unit-testable) ---

const (
	defaultHalfLifeDays = 30.0 // a memory loses half its strength every 30 unused days
	defaultFloor        = 0.1  // decay forgets memories whose strength falls below this
	defaultGuardMin     = 0.5  // guard warns only on memories at least this strong
	defaultGuardOverlap = 2    // guard warns only when at least this many cue tokens overlap
)

func halfLifeDays() float64 {
	if v := os.Getenv("RECOIL_HALFLIFE_DAYS"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return defaultHalfLifeDays
}

// strength is a record's current salience after time-decay: its base salience
// (surprise weight, reinforced by re-fire count) times an exponential decay over
// the days since it was last recalled. Recalling a memory resets that clock and
// raises its hit count, so memories that keep proving useful stay strong while
// untouched ones fade. With halfLife=30, strength halves every 30 unused days.
func strength(r record, now int64, halfLife float64) float64 {
	base := r.Weight * (1 + math.Log(1+float64(r.Hits)))
	ageDays := float64(now-r.Last) / 86400.0
	if ageDays < 0 {
		ageDays = 0
	}
	return base * math.Pow(0.5, ageDays/halfLife)
}

type scored struct {
	idx     int
	score   float64
	matched []string
}

// scoreRecords ranks records by cue overlap with the current situation, weighted
// by current strength — so a faded memory ranks below a fresh one even on an
// equal match. A stored cue is already normalized tokens, so it is split with
// strings.Fields rather than re-tokenized.
func scoreRecords(recs []record, situation map[string]bool, now int64, halfLife float64) []scored {
	var out []scored
	for i, r := range recs {
		var matched []string
		for _, t := range strings.Fields(r.Cue) {
			if situation[t] {
				matched = append(matched, t)
			}
		}
		if len(matched) == 0 {
			continue
		}
		sort.Strings(matched)
		out = append(out, scored{i, float64(len(matched)) * strength(r, now, halfLife), matched})
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].score > out[b].score })
	return out
}

// partitionDecay splits records into those still above the strength floor and
// those that have faded below it.
func partitionDecay(recs []record, now int64, floor, halfLife float64) (keep, forget []record) {
	for _, r := range recs {
		if strength(r, now, halfLife) < floor {
			forget = append(forget, r)
		} else {
			keep = append(keep, r)
		}
	}
	return keep, forget
}

// guardMatches returns the surprise-born memories (not plain notes) that overlap
// the situation on at least minOverlap cue tokens and are still strong enough to
// warn about — the "you've been here before" set, most relevant first. The
// overlap floor keeps a single coincidental shared token (e.g. "unity" across a
// whole domain's lessons) from firing a false warning.
func guardMatches(recs []record, situation map[string]bool, now int64, halfLife, minStrength float64, minOverlap int) []record {
	var out []record
	for _, f := range scoreRecords(recs, situation, now, halfLife) {
		r := recs[f.idx]
		if r.Trigger == "manual" {
			continue // guard is about things that went wrong, not plain notes
		}
		if len(f.matched) < minOverlap {
			continue // too little shared context — likely a coincidental token
		}
		if strength(r, now, halfLife) < minStrength {
			continue
		}
		out = append(out, r)
	}
	return out
}

// --- commands ---

func cmdInit(args []string) {
	if err := os.MkdirAll(storeDir(), 0o755); err != nil {
		die(err)
	}
	if _, err := os.Stat(storePath()); os.IsNotExist(err) {
		if err := saveRecords(nil); err != nil {
			die(err)
		}
	}
	fmt.Printf("recoil: store ready at %s\n", storePath())
}

var triggerWeights = map[string]float64{
	"correction": 3.0,
	"revert":     2.5,
	"test-fail":  2.0,
	"error":      1.5,
	"manual":     1.0,
}

func weightFor(trigger string, override float64) float64 {
	if override >= 0 {
		return override
	}
	if w, ok := triggerWeights[trigger]; ok {
		return w
	}
	return 1.0
}

func cmdEncode(args []string) {
	fs := flag.NewFlagSet("encode", flag.ExitOnError)
	gist := fs.String("gist", "", "the lesson to remember (required)")
	cue := fs.String("cue", "", "situation tokens: files, error text, keywords (required)")
	trigger := fs.String("trigger", "manual", "what surprised the loop: correction|revert|test-fail|error|manual")
	weight := fs.Float64("weight", -1, "salience weight (default: by trigger)")
	fs.Parse(args)

	if *gist == "" || *cue == "" {
		fmt.Fprintln(os.Stderr, "recoil encode: --gist and --cue are required")
		os.Exit(2)
	}
	now := time.Now().Unix()
	r := record{
		ID:      fmt.Sprintf("r%d", time.Now().UnixNano()),
		Created: now,
		Trigger: *trigger,
		Weight:  weightFor(*trigger, *weight),
		Hits:    0,
		Last:    now,
		Cue:     strings.Join(sortedTokens(tokenize(*cue)), " "),
		Gist:    *gist,
	}
	if err := appendRecord(r); err != nil {
		die(err)
	}
	fmt.Printf("recoil: remembered [%s w=%g] %s\n", r.Trigger, r.Weight, r.Gist)
}

// situationFrom builds a situation string from flags, piped stdin, and — if all
// of those are empty and we're in a git repo — the staged files.
func situationFrom(situationFlag, files string, fallbackToStaged bool) map[string]bool {
	var sb strings.Builder
	sb.WriteString(situationFlag)
	sb.WriteByte(' ')
	sb.WriteString(strings.ReplaceAll(files, ",", " "))
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		if b, err := io.ReadAll(os.Stdin); err == nil {
			sb.WriteByte(' ')
			sb.Write(b)
		}
	}
	if fallbackToStaged && strings.TrimSpace(sb.String()) == "" {
		sb.WriteString(stagedFiles())
	}
	return tokenize(sb.String())
}

func cmdRecall(args []string) {
	fs := flag.NewFlagSet("recall", flag.ExitOnError)
	situationFlag := fs.String("situation", "", "describe the current situation")
	files := fs.String("files", "", "comma-separated files in play")
	top := fs.Int("top", 3, "max memories to fire")
	fs.Parse(args)

	situation := situationFrom(*situationFlag, *files, false)
	if len(situation) == 0 {
		fmt.Fprintln(os.Stderr, "recoil recall: no situation given (use --situation, --files, or pipe text)")
		os.Exit(2)
	}

	recs, err := loadRecords()
	if err != nil {
		die(err)
	}
	fired := scoreRecords(recs, situation, time.Now().Unix(), halfLifeDays())
	if len(fired) == 0 {
		fmt.Fprintln(os.Stderr, "recoil: nothing fired (no cue overlap)")
		return
	}

	n := *top
	if n > len(fired) {
		n = len(fired)
	}
	for k := 0; k < n; k++ {
		f := fired[k]
		r := recs[f.idx]
		fmt.Printf(">> %s\n   [%s w=%g hits=%d] matched: %s\n",
			r.Gist, r.Trigger, r.Weight, r.Hits, strings.Join(f.matched, " "))
	}
	reinforce(recs, fired[:n])
}

// reinforce bumps the hit count and last-seen time of the fired records and
// saves. Kept separate from presentation so the ranking is testable without a
// disk write as a side effect. Resetting Last is what keeps a recalled memory
// from fading — using a memory renews it.
func reinforce(recs []record, fired []scored) {
	now := time.Now().Unix()
	for _, f := range fired {
		recs[f.idx].Hits++
		recs[f.idx].Last = now
	}
	if err := saveRecords(recs); err != nil {
		fmt.Fprintln(os.Stderr, "recoil:", err)
	}
}

// cmdGuard warns, before you act, if what you're about to change matches a memory
// of something that went wrong here before (an error, revert, correction, or
// failed test — not a plain note). With no situation given it checks the staged
// files, so it runs as a pre-commit hook. It is read-only: it does not reinforce.
func cmdGuard(args []string) {
	fs := flag.NewFlagSet("guard", flag.ExitOnError)
	files := fs.String("files", "", "comma-separated files you're about to change")
	situationFlag := fs.String("situation", "", "describe what you're about to do")
	minStrength := fs.Float64("min-strength", defaultGuardMin, "only warn on memories at least this strong")
	minOverlap := fs.Int("min-overlap", defaultGuardOverlap, "only warn when at least this many cue tokens overlap")
	block := fs.Bool("block", false, "exit non-zero if a warning fires (abort the action)")
	fs.Parse(args)

	situation := situationFrom(*situationFlag, *files, true)
	if len(situation) == 0 {
		return // nothing to check — stay quiet
	}
	recs, err := loadRecords()
	if err != nil {
		die(err)
	}
	warnings := guardMatches(recs, situation, time.Now().Unix(), halfLifeDays(), *minStrength, *minOverlap)
	for _, r := range warnings {
		fmt.Fprintf(os.Stderr, "recoil: been burned here before — %s\n", r.Gist)
	}
	if len(warnings) > 0 && *block {
		os.Exit(1)
	}
}

// cmdDecay forgets memories whose strength has faded below the floor. Surprise
// weight and re-fire count both raise strength, so a hard-won correction outlives
// a routine note; recalling a memory renews it. --dry-run shows without removing.
func cmdDecay(args []string) {
	fs := flag.NewFlagSet("decay", flag.ExitOnError)
	floor := fs.Float64("floor", defaultFloor, "forget memories whose strength has fallen below this")
	halfLife := fs.Float64("half-life", halfLifeDays(), "days for an unused memory's strength to halve")
	dry := fs.Bool("dry-run", false, "show what would be forgotten without removing it")
	fs.Parse(args)

	recs, err := loadRecords()
	if err != nil {
		die(err)
	}
	keep, forget := partitionDecay(recs, time.Now().Unix(), *floor, *halfLife)
	verb := "forgot"
	if *dry {
		verb = "would forget"
	}
	for _, r := range forget {
		fmt.Printf("%s: [%s w=%g hits=%d] %s\n", verb, r.Trigger, r.Weight, r.Hits, r.Gist)
	}
	if !*dry && len(forget) > 0 {
		if err := saveRecords(keep); err != nil {
			die(err)
		}
	}
	fmt.Fprintf(os.Stderr, "recoil: %d forgotten, %d kept\n", len(forget), len(keep))
}

// cmdWatch runs a command and, if it fails, records the failure as a flinch: the
// command, its error output, and the files in play become the cue, so next time
// you are in a similar spot recoil can remind you it went wrong here. The
// command's own exit code is passed through unchanged.
func cmdWatch(args []string) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "recoil watch: usage: recoil watch -- <command> [args...]")
		os.Exit(2)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	var errBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)

	code := 0
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			fmt.Fprintln(os.Stderr, "recoil watch:", err)
			os.Exit(127)
		}
	}
	if code != 0 {
		recordFailure(args, errBuf.String(), code)
	}
	os.Exit(code)
}

func recordFailure(cmdArgs []string, errOut string, code int) {
	cmdLine := strings.Join(cmdArgs, " ")
	cue := strings.Join(sortedTokens(tokenize(cmdLine+" "+errOut+" "+changedFiles())), " ")
	gist := fmt.Sprintf("`%s` failed (exit %d): %s", cmdLine, code, firstLine(errOut))
	now := time.Now().Unix()
	r := record{
		ID:      fmt.Sprintf("r%d", time.Now().UnixNano()),
		Created: now,
		Trigger: "error",
		Weight:  triggerWeights["error"],
		Hits:    0,
		Last:    now,
		Cue:     cue,
		Gist:    gist,
	}
	if err := appendRecord(r); err != nil {
		fmt.Fprintln(os.Stderr, "recoil:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "recoil: flinch recorded — %s\n", gist)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(no output)"
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// changedFiles returns the names of files git reports as modified, best-effort.
// Outside a git repo it returns "".
func changedFiles() string {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) > 3 {
			b.WriteString(line[3:])
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// stagedFiles lists the files staged for the next commit, best-effort. Outside a
// git repo it returns "".
func stagedFiles() string {
	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(out), "\n", " ")
}

const preCommitHook = `#!/bin/sh
# recoil: warn before repeating a known-bad change (installed by 'recoil hook --install')
recoil guard
`

const postCommitHook = `#!/bin/sh
# recoil: record reverts as flinches (installed by 'recoil hook --install')
subject=$(git log -1 --pretty=%s)
case "$subject" in
  Revert*)
    files=$(git diff-tree --no-commit-id --name-only -r HEAD | tr '\n' ' ')
    recoil encode --trigger revert --gist "$subject" --cue "$subject $files"
    ;;
esac
`

// cmdHook prints or installs recoil's git hooks: a pre-commit that warns before a
// known-bad change, and a post-commit that records reverts. Install never
// overwrites a hook you already have.
func cmdHook(args []string) {
	fs := flag.NewFlagSet("hook", flag.ExitOnError)
	install := fs.Bool("install", false, "write recoil's git hooks (won't overwrite existing ones)")
	fs.Parse(args)

	hooks := []struct{ name, body string }{
		{"pre-commit", preCommitHook},
		{"post-commit", postCommitHook},
	}
	if !*install {
		for _, h := range hooks {
			fmt.Printf("# %s\n%s\n", h.name, h.body)
		}
		return
	}
	for _, h := range hooks {
		out, err := exec.Command("git", "rev-parse", "--git-path", "hooks/"+h.name).Output()
		if err != nil {
			die(fmt.Errorf("not a git repository (run this inside one): %w", err))
		}
		path := strings.TrimSpace(string(out))
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("recoil: %s already exists — not touching it; run `recoil hook` to see the snippet to add\n", h.name)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			die(err)
		}
		if err := os.WriteFile(path, []byte(h.body), 0o755); err != nil {
			die(err)
		}
		fmt.Printf("recoil: installed %s\n", path)
	}
}

func cmdList(args []string) {
	recs, err := loadRecords()
	if err != nil {
		die(err)
	}
	if len(recs) == 0 {
		fmt.Println("recoil: empty")
		return
	}
	now := time.Now().Unix()
	hl := halfLifeDays()
	for _, r := range recs {
		fmt.Printf("[%s w=%g hits=%d str=%.2f] %s\n   cue: %s\n",
			r.Trigger, r.Weight, r.Hits, strength(r, now, hl), r.Gist, r.Cue)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "recoil:", err)
	os.Exit(1)
}

