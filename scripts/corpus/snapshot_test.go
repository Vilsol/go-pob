package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func snapEntry(code, patch string) snapshotBuild {
	sum := sha256.Sum256([]byte(code))
	return snapshotBuild{Code: code, SHA256: hex.EncodeToString(sum[:]), FirstRetainedPeriod: "2026-09", PatchVersion: patch}
}

func snapshotFixture(t *testing.T) (*snapshotFile, string) {
	t.Helper()
	good := decodeFixture(t)
	inCorpus := bytes.Replace(good, []byte("<PathOfBuilding>"), []byte("<PathOfBuilding>\n<!-- corpus -->"), 1)
	if bytes.Equal(inCorpus, good) {
		t.Fatal("fixture has no <PathOfBuilding> line to edit")
	}
	lines := strings.Split(string(inCorpus), "\n")
	resaved := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.Contains(l, "<PlayerStat ") {
			l = strings.Replace(l, `value="`, `value="9`, 1)
		}
		resaved = append(resaved, l)
	}
	resavedX := []byte(strings.Join(resaved, "\r\n"))
	if sha1hex(resavedX) == sha1hex(inCorpus) {
		t.Fatal("the re-saved copy is byte-identical; it cannot test normalisation")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0007-existing.xml"), inCorpus, 0o600); err != nil {
		t.Fatal(err)
	}

	badSha := snapEntry(encodeExport(good), "3.18")
	badSha.SHA256 = strings.Repeat("0", 64)
	s := &snapshotFile{SchemaVersion: 2, SnapshotID: "test-snap", GeneratedAt: "2026-10-03T00:00:00Z", LatestPeriod: "2026-10"}
	s.Builds = []snapshotBuild{
		snapEntry(encodeExport(good), "3.18"),
		badSha,
		snapEntry(encodeExport(good), "3.17"),
		snapEntry("not-a-code!!", "3.18"),
		snapEntry(encodeExport([]byte("plain text, not a build")), "3.18"),
		snapEntry(encodeExport(resavedX), "3.18"),
		snapEntry(encodeExport(good), "3.18"),
		snapEntry(encodeExport(bytes.ReplaceAll(good, []byte(`treeVersion="3_18"`), []byte(`treeVersion="3_17"`))), "3.18"),
	}
	s.Count = len(s.Builds)
	return s, dir
}

func TestSnapshotRefusesAndCountsEachClass(t *testing.T) {
	s, dir := snapshotFixture(t)
	rows, err := importSnapshot(s, snapshotOpts{corpusDir: dir, patch: "3.18", crit: criteria318()})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"accepted", rejSHA256Mismatch, rejWrongPatch, rejDecodeFailed, rejXMLParseFailed,
		rejDuplicate, rejDuplicateSnapshot, rejWrongTree,
	}
	if len(rows) != len(want) {
		t.Fatalf("%d rows for %d codes", len(rows), len(want))
	}
	for i, w := range want {
		if rows[i].Verdict != w {
			t.Errorf("code %d: verdict %q (%s), want %q", i, rows[i].Verdict, rows[i].Detail, w)
		}
	}
	if rows[5].Detail != "0007-existing.xml" {
		t.Errorf("the corpus duplicate names %q, want the existing file", rows[5].Detail)
	}
	if !strings.HasPrefix(rows[0].File, "0008-") || rows[0].xml == nil {
		t.Errorf("the accepted build is %q; it must continue after the corpus's last number", rows[0].File)
	}
}

func TestSnapshotImportIsIdempotent(t *testing.T) {
	s, dir := snapshotFixture(t)
	work := t.TempDir()
	in := filepath.Join(work, "snap.json")
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in, b, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "MANIFEST.md")
	if err := os.WriteFile(manifest, []byte("# corpus\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-in", in, "-corpus", dir, "-patch", "3.18", "-tree", "3_18", "-index", filepath.Join(work, "index"), "-write"}

	count := func() (int, string) {
		files, _ := filepath.Glob(filepath.Join(dir, "*.xml"))
		m, _ := os.ReadFile(manifest) //nolint:gosec // a test temp file
		return len(files), string(m)
	}
	if err := cmdSnapshot(args); err != nil {
		t.Fatal(err)
	}
	n1, m1 := count()
	if n1 != 2 || !strings.Contains(m1, "`0008-") {
		t.Fatalf("first run: %d files, manifest:\n%s", n1, m1)
	}
	if err := cmdSnapshot(args); err != nil {
		t.Fatal(err)
	}
	if n2, m2 := count(); n2 != n1 || m2 != m1 {
		t.Fatalf("second run changed the corpus: %d -> %d files, manifest grew %d bytes", n1, n2, len(m2)-len(m1))
	}
}
