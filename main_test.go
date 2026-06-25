package main

import "testing"

const day = int64(86400)

func TestTokenizeDropsStopwordsAndSplits(t *testing.T) {
	got := tokenize("Build/ folder in the .gitignore")
	for _, want := range []string{"build", "folder", "gitignore"} {
		if !got[want] {
			t.Errorf("expected token %q", want)
		}
	}
	if got["in"] || got["the"] {
		t.Error("stopwords should be dropped")
	}
}

func TestRecordRoundTrip(t *testing.T) {
	r := record{
		ID: "r1", Created: 100, Trigger: "error", Weight: 1.5,
		Hits: 2, Last: 200, Cue: "alpha beta", Gist: "a lesson",
	}
	got, ok := parseRecord(r.tsv())
	if !ok {
		t.Fatal("parseRecord rejected its own output")
	}
	if got != r {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, r)
	}
}

func TestParseRecordRejectsGarbage(t *testing.T) {
	if _, ok := parseRecord("too\tfew\tcolumns"); ok {
		t.Error("expected failure on wrong column count")
	}
	// 8 columns but a non-numeric weight — must be rejected, not loaded as zero
	if _, ok := parseRecord("r1\t100\terror\tNOTNUM\t0\t200\tcue\tgist"); ok {
		t.Error("expected failure on non-numeric field")
	}
}

func TestScoreRecordsFiresAndIgnores(t *testing.T) {
	recs := []record{
		{Trigger: "correction", Weight: 3, Cue: "menu options proposal"},
		{Trigger: "test-fail", Weight: 2, Cue: "unity build gitignore"},
	}
	fired := scoreRecords(recs, tokenize("about to show a menu of options"), 0, defaultHalfLifeDays)
	if len(fired) != 1 || fired[0].idx != 0 {
		t.Fatalf("expected only the correction record to fire, got %+v", fired)
	}
	if got := scoreRecords(recs, tokenize("tuning the water shader foam"), 0, defaultHalfLifeDays); len(got) != 0 {
		t.Errorf("unrelated situation should fire nothing, got %d", len(got))
	}
}

func TestScoreRecordsRanksBySalience(t *testing.T) {
	// same single-token overlap and same age, but the higher-weight record wins
	recs := []record{
		{Trigger: "manual", Weight: 1, Cue: "alpha"},
		{Trigger: "correction", Weight: 3, Cue: "alpha"},
	}
	fired := scoreRecords(recs, tokenize("alpha"), 0, defaultHalfLifeDays)
	if len(fired) != 2 || fired[0].idx != 1 {
		t.Fatalf("expected the weight=3 record first, got %+v", fired)
	}
}

func TestStrengthHalvesAtHalfLife(t *testing.T) {
	r := record{Weight: 2, Hits: 0, Last: 0}
	fresh := strength(r, 0, 30)
	halved := strength(r, 30*day, 30)
	if fresh <= 0 {
		t.Fatal("fresh strength should be positive")
	}
	if ratio := halved / fresh; ratio < 0.49 || ratio > 0.51 {
		t.Errorf("expected ~0.5 after one half-life, got %.3f", ratio)
	}
}

func TestScoreRecordsPrefersFreshOverStale(t *testing.T) {
	now := 100 * day
	recs := []record{
		{Weight: 2, Cue: "alpha", Last: now - 90*day}, // long unused
		{Weight: 2, Cue: "alpha", Last: now},          // just used
	}
	fired := scoreRecords(recs, tokenize("alpha"), now, 30)
	if len(fired) != 2 || fired[0].idx != 1 {
		t.Fatalf("the fresh record should rank first, got %+v", fired)
	}
}

func TestPartitionDecayForgetsFaded(t *testing.T) {
	now := 1000 * day
	recs := []record{
		{Gist: "fresh", Weight: 1, Last: now},             // strong
		{Gist: "ancient", Weight: 1, Last: now - 365*day}, // faded well below floor
	}
	keep, forget := partitionDecay(recs, now, defaultFloor, defaultHalfLifeDays)
	if len(keep) != 1 || keep[0].Gist != "fresh" {
		t.Errorf("expected to keep only 'fresh', kept %+v", keep)
	}
	if len(forget) != 1 || forget[0].Gist != "ancient" {
		t.Errorf("expected to forget 'ancient', forgot %+v", forget)
	}
}
