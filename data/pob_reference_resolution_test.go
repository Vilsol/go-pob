package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReferenceResolutionRefusals exercises the arms of resolvePoBTableReference that NEVER FIRE IN A HEALTHY TREE.
func TestReferenceResolutionRefusals(t *testing.T) {
	const (
		pinnedHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		otherHash  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	want := pobPinRecord{ContentSHA256: pinnedHash, Describe: "v9.9.9-pinned"}

	write := func(t *testing.T, dir, name, hash, describe string) {
		t.Helper()
		blob, err := json.Marshal(pobTableDump{
			PoBVersion:  "9.9.9",
			TreeVersion: "9_9",
			PoBPin:      pobPinRecord{ContentSHA256: hash, DescribeAdvisory: describe, FileCount: 7},
			Tables:      map[string]pobTable{"keystones": {Kind: "stringlist"}},
		})
		if err != nil {
			t.Fatalf("marshalling the fixture failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), blob, 0o600); err != nil {
			t.Fatalf("writing the fixture failed: %v", err)
		}
	}

	t.Run("a reference for another pin is REFUSED, not used", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "pob-data-tables-3_20.json", otherHash, "v2.25.1-99-gfddbbc8f")

		dump, path, err := resolvePoBTableReference(dir, want)
		if err == nil {
			t.Fatalf("RESOLVED %s instead of refusing. The reference belongs to pin %s and the "+
				"tree pins %s — using it would grade the port against the PREVIOUS version's "+
				"tables and report no drift. dump=%v", path, otherHash, pinnedHash, dump != nil)
		}
		for _, want := range []string{"no reference", pinnedHash, "pob-data-tables-3_20.json"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("refusal message does not mention %q; a refusal that cannot be acted on "+
					"is only marginally better than a wrong answer.\ngot: %v", want, err)
			}
		}
	})

	t.Run("two references claiming one pin are REFUSED", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "pob-data-tables-3_20.json", pinnedHash, "v9.9.9-pinned")
		write(t, dir, "pob-data-tables-copy.json", pinnedHash, "v9.9.9-pinned")

		if _, path, err := resolvePoBTableReference(dir, want); err == nil {
			t.Fatalf("RESOLVED %s with two candidates. Picking either makes the verdict depend "+
				"on directory order, which is a nondeterministic gate on a project whose whole "+
				"instrument set exists to remove nondeterminism.", path)
		}
	})

	t.Run("a reference with no recorded pin never matches", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "pob-data-tables-legacy.json", "", "")

		if _, _, err := resolvePoBTableReference(dir, want); err == nil {
			t.Error("RESOLVED a reference carrying no pin. A reference that cannot name its own " +
				"checkout is exactly the state this field was added to eliminate.")
		}
		if _, _, err := resolvePoBTableReference(dir, pobPinRecord{}); err == nil {
			t.Error("an empty pin record matched a reference with an empty pin — two unknowns " +
				"comparing equal, which would make every reference authoritative at once")
		}
	})

	t.Run("the matching reference IS resolved", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "pob-data-tables-3_20.json", otherHash, "v-other")
		write(t, dir, "pob-data-tables-9_9.json", pinnedHash, "v9.9.9-pinned")

		dump, path, err := resolvePoBTableReference(dir, want)
		if err != nil {
			t.Fatalf("refused a reference that DOES match the pin: %v", err)
		}
		if filepath.Base(path) != "pob-data-tables-9_9.json" {
			t.Errorf("resolved %s, want pob-data-tables-9_9.json — it picked by name or by order "+
				"rather than by pin", filepath.Base(path))
		}
		if dump.PoBPin.ContentSHA256 != pinnedHash {
			t.Errorf("resolved a dump recording pin %s, want %s", dump.PoBPin.ContentSHA256, pinnedHash)
		}
	})

	t.Run("an UNSTAMPED reference never matches", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "pob-data-tables-9_9.json", "", "")

		if _, _, err := resolvePoBTableReference(dir, want); err == nil {
			t.Error("RESOLVED an unstamped reference")
		}
	})
}

// TestRecordedPinIsUsable pins the assumption the resolver rests on: the repo's own pin record parses into the shape used here and carries a content hash.
func TestRecordedPinIsUsable(t *testing.T) {
	blob, err := os.ReadFile(recordedPinPath)
	if err != nil {
		t.Fatalf("REFUSING: cannot read %s: %v", recordedPinPath, err)
	}
	var pin pobPinRecord
	if err := json.Unmarshal(blob, &pin); err != nil {
		t.Fatalf("REFUSING: %s is not valid JSON: %v", recordedPinPath, err)
	}
	if len(pin.ContentSHA256) != 64 {
		t.Errorf("%s records content_sha256 %q, which is not a 64-character sha256 digest. The "+
			"resolver matches references on this value exactly.", recordedPinPath, pin.ContentSHA256)
	}
	if pin.Describe == "" {
		t.Errorf("%s records no \"describe\". It is what the refusal message shows the reader, "+
			"and a bare sha is far harder to act on.", recordedPinPath)
	}
}
