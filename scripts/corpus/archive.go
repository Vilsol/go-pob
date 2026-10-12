package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ArchiveRow struct {
	URL       string `json:"url"`
	Host      string `json:"host"`
	Source    string `json:"source"`
	PostedAt  string `json:"posted_at"`
	Context   string `json:"context,omitempty"`
	FirstSeen string `json:"first_seen,omitempty"`

	LastStatus  string `json:"last_fetch_status"`
	LastFetchAt string `json:"last_fetch_at,omitempty"`
	HTTP        int    `json:"http,omitempty"`
	Via         string `json:"via,omitempty"`

	Verdict string `json:"verdict"`
	Detail  string `json:"detail,omitempty"`
	File    string `json:"file,omitempty"`
	SHA1    string `json:"sha1,omitempty"`
}

const archiveRoot = "testdata/corpus-index"

func archivePath(repo, league string) string {
	return filepath.Join(repo, archiveRoot, league, "index.jsonl.gz")
}

func cmdArchive(args []string) error {
	fs := flag.NewFlagSet("archive", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory for all stage artifacts")
	repo := fs.String("repo", ".", "repository root; the archive lands under <repo>/"+archiveRoot)
	league := fs.String("league", "", "league key for the archive path, e.g. 3.19 (required)")
	out := fs.String("out", "", "override the output path entirely")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	if *league == "" && *out == "" {
		return fmt.Errorf("-league is required (e.g. -league 3.19); refusing to guess an archive path")
	}

	idx, err := loadIndex(*work)
	if err != nil {
		return err
	}
	if len(idx.byID) == 0 {
		return fmt.Errorf("link index under %s is empty — nothing to archive, and an empty archive would read as an exhausted source", *work)
	}
	fetched, err := loadFetchLog(*work)
	if err != nil {
		return err
	}
	accepted, err := readJSONL[Accepted](filepath.Join(*work, "builds.jsonl"))
	if err != nil {
		return err
	}
	rejects, err := readJSONL[Reject](filepath.Join(*work, "rejects.jsonl"))
	if err != nil {
		return err
	}

	acceptedBy := map[string]Accepted{}
	for _, a := range accepted {
		acceptedBy[a.URL] = a
	}
	rejectBy := map[string]Reject{}
	for _, r := range rejects {
		rejectBy[r.URL] = r
	}

	rows := make([]ArchiveRow, 0, len(idx.byID))
	for _, l := range idx.links() {
		row := ArchiveRow{
			URL: l.URL, Host: l.Host, Source: l.Source, PostedAt: l.PostedAt,
			Context: l.Context, FirstSeen: l.FirstSeen, LastStatus: "not_fetched",
			Verdict: "not_validated",
		}
		if f, ok := fetched[l.URL]; ok {
			row.LastStatus, row.LastFetchAt, row.HTTP, row.Via = string(f.Class), f.At, f.HTTP, f.Via
		}
		if a, ok := acceptedBy[l.URL]; ok {
			row.Verdict, row.File, row.SHA1 = "accepted", a.File, a.SHA1
		} else if r, ok := rejectBy[l.URL]; ok {
			row.Verdict, row.Detail = r.Reason, r.Detail
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].URL < rows[j].URL })

	path := *out
	if path == "" {
		path = archivePath(*repo, *league)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	zw, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		f.Close()
		return fmt.Errorf("gzip: %w", err)
	}
	zw.ModTime = time.Time{}
	w := bufio.NewWriter(zw)
	enc := json.NewEncoder(w)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			f.Close()
			return fmt.Errorf("encoding %s: %w", r.URL, err)
		}
	}
	for _, closeErr := range []error{w.Flush(), zw.Close(), f.Close()} {
		if closeErr != nil {
			return fmt.Errorf("writing %s: %w", path, closeErr)
		}
	}

	st, _ := os.Stat(path)
	banner("archive: %d candidates -> %s (%d KB)", len(rows), path, st.Size()/1024)
	summariseArchive(rows)
	return nil
}

func summariseArchive(rows []ArchiveRow) {
	byStatus := map[string]int{}
	byVerdict := map[string]int{}
	bySource := map[string]int{}
	for _, r := range rows {
		byStatus[r.LastStatus]++
		byVerdict[r.Verdict]++
		bySource[r.Source]++
	}
	fmt.Println("\nlast_fetch_status:")
	for _, r := range sortMap(byStatus) {
		fmt.Printf("  %-22s %6d\n", r.Key, r.N)
	}
	fmt.Println("verdict:")
	for _, r := range sortMap(byVerdict) {
		fmt.Printf("  %-22s %6d\n", r.Key, r.N)
	}
	fmt.Println("source:")
	for _, r := range sortMap(bySource) {
		fmt.Printf("  %-22s %6d\n", r.Key, r.N)
	}
}

func cmdSeed(args []string) error {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory to seed")
	repo := fs.String("repo", ".", "repository root holding "+archiveRoot)
	league := fs.String("league", "", "league key, e.g. 3.19 (required unless -in is given)")
	in := fs.String("in", "", "read this archive path instead of deriving it from -league")
	withFetch := fs.Bool("with-fetch", false,
		"also seed the fetch log, so links already resolved are not re-requested (⚠️ the raw blobs are NOT in the archive, so validate will report them unreachable until re-fetched)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	path := *in
	if path == "" {
		if *league == "" {
			return fmt.Errorf("-league or -in is required")
		}
		path = archivePath(*repo, *league)
	}

	rows, err := readArchive(path)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("%s holds no rows — refusing to seed an empty index, which a later run would read as an exhausted source", path)
	}

	idx, err := loadIndex(*work)
	if err != nil {
		return err
	}
	added := 0
	for _, r := range rows {
		if idx.add(Link{URL: r.URL, Host: r.Host, Source: r.Source, PostedAt: r.PostedAt,
			Context: r.Context, FirstSeen: r.FirstSeen}) {
			added++
		}
	}
	if err := idx.save(); err != nil {
		return err
	}

	banner("seed: %d rows from %s -> %d new, %d total in %s", len(rows), path, added, len(idx.byID), idx.path)

	if *withFetch {
		logPath := filepath.Join(*work, "fetch.jsonl")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("opening %s: %w", logPath, err)
		}
		enc := json.NewEncoder(f)
		n := 0
		for _, r := range rows {
			if r.LastStatus == "" || r.LastStatus == "not_fetched" {
				continue
			}
			if err := enc.Encode(FetchRecord{URL: r.URL, Host: r.Host, Class: fetchClass(r.LastStatus),
				HTTP: r.HTTP, Via: r.Via, At: r.LastFetchAt, Note: "seeded from " + filepath.Base(path)}); err != nil {
				f.Close()
				return fmt.Errorf("writing %s: %w", logPath, err)
			}
			n++
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("closing %s: %w", logPath, err)
		}
		fmt.Printf("seeded %d fetch records — the raw blobs are NOT archived, so re-run `fetch -retry-blocked=false`\n", n)
		fmt.Println("only if you still hold raw/; otherwise clear fetch.jsonl and re-fetch.")
	}
	return nil
}

func cmdLinkrot(args []string) error {
	fs := flag.NewFlagSet("linkrot", flag.ExitOnError)
	in := fs.String("in", "", "archive path to sample (required)")
	work := fs.String("work", "", "scratch working directory for the sample (required)")
	sample := fs.Int("sample", 300, "how many links to re-check (0 = all)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	if *in == "" || *work == "" {
		return fmt.Errorf("-in and -work are both required")
	}
	rows, err := readArchive(*in)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("%s holds no rows", *in)
	}
	if err := os.MkdirAll(*work, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", *work, err)
	}
	stride := 1
	if *sample > 0 && len(rows) > *sample {
		stride = len(rows) / *sample
	}
	idx := &linkIndex{path: filepath.Join(*work, "index.jsonl"), byID: map[string]Link{}}
	for i := 0; i < len(rows); i += stride {
		r := rows[i]
		idx.add(Link{URL: r.URL, Host: r.Host, Source: r.Source, PostedAt: r.PostedAt,
			Context: r.Context, FirstSeen: r.FirstSeen})
	}
	if err := idx.save(); err != nil {
		return err
	}
	banner("linkrot: sampled %d of %d rows from %s", len(idx.byID), len(rows), *in)
	fmt.Printf("now run:  go run ./scripts/corpus fetch -work %s\n", *work)
	fmt.Println("and read the class table — `notfound` is rot, `blocked` is us being throttled.")
	return nil
}

func readArchive(path string) ([]ArchiveRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gunzip %s: %w", path, err)
	}
	defer zr.Close()

	var out []ArchiveRow
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for ln := 1; sc.Scan(); ln++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r ArchiveRow
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, ln, err)
		}
		out = append(out, r)
	}
	return out, wrapScan(sc.Err())
}
