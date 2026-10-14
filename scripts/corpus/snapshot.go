package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	rejSHA256Mismatch     = "sha256_mismatch"
	rejWrongPatch         = "wrong_patch"
	rejDuplicateSnapshot  = "duplicate_in_snapshot"
	snapshotSchemaVersion = 2
)

type snapshotBuild struct {
	Code                string `json:"code"`
	SHA256              string `json:"sha256"`
	FirstRetainedPeriod string `json:"firstRetainedPeriod"`
	PatchVersion        string `json:"patchVersion"`
}

type snapshotFile struct {
	SchemaVersion int             `json:"schemaVersion"`
	SnapshotID    string          `json:"snapshotId"`
	GeneratedAt   string          `json:"generatedAt"`
	LatestPeriod  string          `json:"latestPeriod"`
	Count         int             `json:"count"`
	Builds        []snapshotBuild `json:"builds"`
}

type SnapshotRow struct {
	SnapshotID          string `json:"snapshot_id"`
	Index               int    `json:"index"`
	SHA256              string `json:"sha256"`
	FirstRetainedPeriod string `json:"first_retained_period"`
	PatchVersion        string `json:"patch_version"`
	Verdict             string `json:"verdict"`
	Detail              string `json:"detail,omitempty"`
	File                string `json:"file,omitempty"`
	SHA1                string `json:"sha1,omitempty"`
	NormSHA1            string `json:"norm_sha1,omitempty"`

	xml []byte
}

type snapshotOpts struct {
	corpusDir string
	patch     string
	crit      criteria
}

func readSnapshot(path string) (*snapshotFile, error) {
	b, err := os.ReadFile(path) //nolint:gosec // an operator-supplied snapshot
	if err != nil {
		return nil, fmt.Errorf("readSnapshot: %w", err)
	}
	var s snapshotFile
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	if s.SchemaVersion != snapshotSchemaVersion {
		return nil, fmt.Errorf("%s: schemaVersion %d, this importer reads %d", path, s.SchemaVersion, snapshotSchemaVersion)
	}
	if s.SnapshotID == "" || len(s.Builds) == 0 || s.Count != len(s.Builds) {
		return nil, fmt.Errorf("%s: snapshotId %q, count %d, %d builds — refusing a partial snapshot",
			path, s.SnapshotID, s.Count, len(s.Builds))
	}
	return &s, nil
}

var reNormLine = regexp.MustCompile(`^\s*<(PlayerStat|MinionStat) `)

func normXMLSHA1(x []byte) string {
	x = bytes.ReplaceAll(x, []byte("\r\n"), []byte("\n"))
	var b bytes.Buffer
	for _, l := range bytes.Split(x, []byte("\n")) {
		if reNormLine.Match(l) {
			continue
		}
		b.Write(bytes.TrimRight(l, " \t\r"))
		b.WriteByte('\n')
	}
	return sha1hex(bytes.TrimSpace(b.Bytes()))
}

var reCorpusNumber = regexp.MustCompile(`^(\d+)-`)

func scanCorpus(dir string) (map[string]string, int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return nil, 0, fmt.Errorf("scanCorpus: %w", err)
	}
	if len(files) == 0 {
		return nil, 0, fmt.Errorf("no *.xml under %s — refusing to dedupe against an empty corpus", dir)
	}
	seen := make(map[string]string, len(files))
	maxN := 0
	for _, f := range files {
		x, err := os.ReadFile(f) //nolint:gosec // a corpus file
		if err != nil {
			return nil, 0, fmt.Errorf("scanCorpus: %w", err)
		}
		if bytes.HasPrefix(x, []byte("version https://git-lfs")) {
			return nil, 0, fmt.Errorf("%s is a git-lfs pointer — run `git lfs pull` first", f)
		}
		seen[normXMLSHA1(x)] = filepath.Base(f)
		if m := reCorpusNumber.FindStringSubmatch(filepath.Base(f)); m != nil {
			if n, _ := strconv.Atoi(m[1]); n > maxN {
				maxN = n
			}
		}
	}
	return seen, maxN, nil
}

func importSnapshot(s *snapshotFile, opt snapshotOpts) ([]SnapshotRow, error) {
	seen, next, err := scanCorpus(opt.corpusDir)
	if err != nil {
		return nil, err
	}
	inSnapshot := map[string]int{}
	rows := make([]SnapshotRow, 0, len(s.Builds))
	for i, b := range s.Builds {
		r := SnapshotRow{
			SnapshotID: s.SnapshotID, Index: i, SHA256: b.SHA256,
			FirstRetainedPeriod: b.FirstRetainedPeriod, PatchVersion: b.PatchVersion,
		}
		rows = append(rows, judgeSnapshotBuild(r, b, opt, seen, inSnapshot, &next))
	}
	return rows, nil
}

func judgeSnapshotBuild(r SnapshotRow, b snapshotBuild, opt snapshotOpts, seen map[string]string, inSnapshot map[string]int, next *int) SnapshotRow {
	sum := sha256.Sum256([]byte(b.Code))
	if hex.EncodeToString(sum[:]) != strings.ToLower(b.SHA256) {
		r.Verdict = rejSHA256Mismatch
		return r
	}
	if b.PatchVersion != opt.patch {
		r.Verdict, r.Detail = rejWrongPatch, b.PatchVersion
		return r
	}
	x, err := decodeExport([]byte(b.Code))
	if err != nil {
		r.Verdict, r.Detail = rejDecodeFailed, err.Error()
		return r
	}
	if !isPoBXML(x) {
		r.Verdict = rejXMLParseFailed
		r.Detail = "no <PathOfBuilding> element"
		return r
	}
	v := judge(x, opt.crit)
	r.SHA1, r.NormSHA1 = sha1hex(x), normXMLSHA1(x)
	if !v.ok {
		r.Verdict, r.Detail = v.reason, v.detail
		return r
	}
	if f, ok := seen[r.NormSHA1]; ok {
		r.Verdict, r.Detail = rejDuplicate, f
		return r
	}
	if j, ok := inSnapshot[r.NormSHA1]; ok {
		r.Verdict, r.Detail = rejDuplicateSnapshot, "index "+strconv.Itoa(j)
		return r
	}
	inSnapshot[r.NormSHA1] = r.Index
	*next++
	r.Verdict = "accepted"
	r.File = fmt.Sprintf("%04d-%s-%s.xml", *next, v.doc.classLabel(), v.doc.mainSkillLabel())
	r.xml = x
	return r
}

func cmdSnapshot(args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	in := fs.String("in", "", "the snapshot JSON (required)")
	corpusDir := fs.String("corpus", "", "the corpus directory to dedupe against and write into (required)")
	patch := fs.String("patch", "", "required patchVersion, e.g. 3.29 (required)")
	tree := fs.String("tree", "", "required ACTIVE tree version, e.g. 3_29 (required)")
	indexDir := fs.String("index", "", "directory for the per-snapshot verdict file (required)")
	manifest := fs.String("manifest", "", "MANIFEST.md to append the addendum to (default <corpus>/MANIFEST.md)")
	write := fs.Bool("write", false, "write the accepted builds, the verdict file and the addendum (default: dry run)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	if *in == "" || *corpusDir == "" || *patch == "" || *tree == "" || *indexDir == "" {
		return fmt.Errorf("-in, -corpus, -patch, -tree and -index are all required")
	}
	if *manifest == "" {
		*manifest = filepath.Join(*corpusDir, "MANIFEST.md")
	}

	s, err := readSnapshot(*in)
	if err != nil {
		return err
	}
	verdicts := filepath.Join(*indexDir, "pobcodes-"+s.SnapshotID+".jsonl")
	if _, err := os.Stat(verdicts); err == nil {
		return verifyImportedSnapshot(verdicts, *corpusDir)
	}

	banner("snapshot %s (%s, %d codes): patch=%s tree=%s", s.SnapshotID, s.GeneratedAt, len(s.Builds), *patch, *tree)
	opt := snapshotOpts{
		corpusDir: *corpusDir, patch: *patch,
		crit: criteria{treeVersion: *tree, minNodes: 80, minSlots: 6, slotMode: "occurrences"},
	}
	rows, err := importSnapshot(s, opt)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.Verdict]++
	}
	for _, c := range sortMap(counts) {
		fmt.Printf("  %-22s %6d\n", c.Key, c.N)
	}
	if !*write {
		fmt.Println("\ndry run — pass -write to import")
		return nil
	}

	var files []string
	for _, r := range rows {
		if r.xml == nil {
			continue
		}
		p := filepath.Join(*corpusDir, r.File)
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("%s already exists — refusing to overwrite", p)
		}
		if err := os.WriteFile(p, r.xml, 0o644); err != nil { //nolint:gosec // corpus files are world-readable data
			return fmt.Errorf("cmdSnapshot: %w", err)
		}
		files = append(files, r.File)
	}
	if err := os.MkdirAll(*indexDir, 0o755); err != nil { //nolint:gosec // a repo directory
		return fmt.Errorf("cmdSnapshot: %w", err)
	}
	if err := writeJSONL(verdicts, rows); err != nil {
		return err
	}
	if err := appendSnapshotAddendum(*manifest, s, rows, verdicts, counts); err != nil {
		return err
	}
	fmt.Printf("\nwrote %d builds, %s, and an addendum to %s\n", len(files), verdicts, *manifest)
	return nil
}

func verifyImportedSnapshot(verdicts, corpusDir string) error {
	rows, err := readJSONL[SnapshotRow](verdicts)
	if err != nil {
		return err
	}
	banner("snapshot already imported (%s): verifying", verdicts)
	accepted := 0
	for _, r := range rows {
		if r.Verdict != "accepted" {
			continue
		}
		accepted++
		x, err := os.ReadFile(filepath.Join(corpusDir, r.File)) //nolint:gosec // a corpus file
		if err != nil {
			return fmt.Errorf("verifyImportedSnapshot: %w", err)
		}
		if sha1hex(x) != r.SHA1 {
			return fmt.Errorf("%s no longer matches sha1 %s recorded in %s", r.File, r.SHA1, verdicts)
		}
	}
	fmt.Printf("%d rows, %d accepted files present and unchanged; nothing imported\n", len(rows), accepted)
	return nil
}

func appendSnapshotAddendum(path string, s *snapshotFile, rows []SnapshotRow, verdicts string, counts map[string]int) error {
	var b strings.Builder
	var first, last string
	for _, r := range rows {
		if r.File == "" {
			continue
		}
		if first == "" {
			first = r.File
		}
		last = r.File
	}
	num := func(f string) string { return reCorpusNumber.FindStringSubmatch(f)[1] }
	if first == "" {
		fmt.Fprintf(&b, "\n## Addendum %s — pob.codes snapshot `%s`: no builds accepted\n\n",
			time.Now().UTC().Format("2006-01-02"), s.SnapshotID)
	} else {
		fmt.Fprintf(&b, "\n## Addendum %s — %d pob.codes test builds (`%s`..`%s`)\n\n",
			time.Now().UTC().Format("2006-01-02"), counts["accepted"], num(first), num(last))
	}
	fmt.Fprintf(&b, "Source: `https://api.pob.codes/test-builds/corpus`, snapshot `%s` (generated %s, latest period %s), "+
		"%d codes, imported with `scripts/corpus snapshot`. Each code's `sha256` is the SHA-256 of the export code string as "+
		"served and was verified before decoding. Judged with the SAME acceptance criteria as above, then deduplicated "+
		"against the corpus and the snapshot by SHA-1 of the XML with LF line ends, trailing whitespace stripped and "+
		"`<PlayerStat>`/`<MinionStat>` lines dropped. Every code's verdict: `%s`.\n\n",
		s.SnapshotID, s.GeneratedAt, s.LatestPeriod, len(rows), filepath.Base(verdicts))
	b.WriteString("| verdict | count |\n|---|---|\n")
	for _, c := range sortMap(counts) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", c.Key, c.N)
	}
	b.WriteString("\n")
	for _, r := range rows {
		if r.File != "" {
			fmt.Fprintf(&b, "- `%s` — period `%s`, sha1 `%s`\n", r.File, r.FirstRetainedPeriod, r.SHA1)
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0) //nolint:gosec // an operator-supplied manifest
	if err != nil {
		return fmt.Errorf("appendSnapshotAddendum: %w", err)
	}
	w := bufio.NewWriter(f)
	if _, err := w.WriteString(b.String()); err != nil {
		_ = f.Close()
		return fmt.Errorf("appendSnapshotAddendum: %w", err)
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return fmt.Errorf("appendSnapshotAddendum: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("appendSnapshotAddendum: %w", err)
	}
	return nil
}
