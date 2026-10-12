package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Reject struct {
	URL    string `json:"url"`
	Source string `json:"source"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

type Accepted struct {
	File     string `json:"file"`
	URL      string `json:"url"`
	Host     string `json:"host"`
	Source   string `json:"source"`
	PostedAt string `json:"posted_at"`
	Context  string `json:"context"`
	Class    string `json:"class"`
	Ascend   string `json:"ascend"`
	Skill    string `json:"skill"`
	Level    int    `json:"level"`
	Nodes    int    `json:"nodes"`
	Slots    int    `json:"slots"`
	Stats    int    `json:"stats"`
	SHA1     string `json:"sha1"`
}

func cmdValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory for all stage artifacts")
	out := fs.String("out", "", "directory to write accepted build XML into (default <work>/builds)")
	tree := fs.String("tree", "", "required ACTIVE tree version, e.g. 3_19 (required)")
	minNodes := fs.Int("min-nodes", 80, "minimum allocated passives in the active spec")
	listRejects := fs.Bool("list-rejects", false, "-corpus: print EVERY rejected path, not the 15-line sample")
	minSlots := fs.Int("min-slots", 6, "minimum filled equipment slots")
	slotMode := fs.String("slot-mode", "occurrences", "how to count filled slots: occurrences (3.18-compatible) | distinct")
	corpusDir := fs.String("corpus", "", "grade an EXISTING directory of build XML instead of the fetch log (control mode)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}
	if *tree == "" {
		return fmt.Errorf("-tree is required (e.g. -tree 3_19); refusing to guess a league")
	}

	c := criteria{treeVersion: *tree, minNodes: *minNodes, minSlots: *minSlots, slotMode: *slotMode}

	if *corpusDir != "" {
		return validateExistingCorpus(*corpusDir, c, *listRejects)
	}

	outDir := *out
	if outDir == "" {
		outDir = filepath.Join(*work, "builds")
	}

	idx, err := loadIndex(*work)
	if err != nil {
		return err
	}
	fetched, err := loadFetchLog(*work)
	if err != nil {
		return err
	}
	if len(fetched) == 0 {
		return fmt.Errorf("no fetch log under %s — run `fetch` first; refusing to report 0 accepted from an empty input", *work)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", outDir, err)
	}

	banner("validate: tree=%s minNodes=%d minSlots=%d (%s)", *tree, *minNodes, *minSlots, *slotMode)

	var (
		reasons  = map[string]int{}
		wrongVer = map[string]int{}
		accepted = make([]Accepted, 0, 256)
		rejects  []Reject
		bySha    = map[string]string{}
		nameSeq  = map[string]int{}
		rawDir   = filepath.Join(*work, "raw")
	)

	links := idx.links()
	sortLinksForNumbering(links)

	type pending struct {
		l Link
		v verdict
		x []byte
	}
	pass := make([]pending, 0, len(links))

	for _, l := range links {
		rec, ok := fetched[l.URL]
		if !ok {
			reasons[rejUnreachable]++
			rejects = append(rejects, Reject{l.URL, l.Source, rejUnreachable, "never attempted"})
			continue
		}
		if rec.Class != fetchOK {
			reasons[rejUnreachable]++
			rejects = append(rejects, Reject{l.URL, l.Source, rejUnreachable, string(rec.Class)})
			continue
		}
		blob, err := os.ReadFile(filepath.Join(rawDir, rec.Blob))
		if err != nil {
			reasons[rejUnreachable]++
			rejects = append(rejects, Reject{l.URL, l.Source, rejUnreachable, "blob missing: " + err.Error()})
			continue
		}
		x, err := decodeExport(blob)
		if err != nil {
			reasons[rejDecodeFailed]++
			rejects = append(rejects, Reject{l.URL, l.Source, rejDecodeFailed, err.Error()})
			continue
		}
		v := judge(x, c)
		if !v.ok {
			reasons[v.reason]++
			if v.reason == rejWrongTree {
				wrongVer[orUnknown(v.detail)]++
			}
			rejects = append(rejects, Reject{l.URL, l.Source, v.reason, v.detail})
			continue
		}
		pass = append(pass, pending{l, v, x})
	}

	for _, p := range pass {
		if prev, dup := bySha[p.v.xmlSha1]; dup {
			reasons[rejDuplicate]++
			rejects = append(rejects, Reject{p.l.URL, p.l.Source, rejDuplicate, "same XML as " + prev})
			continue
		}
		base := fmt.Sprintf("%s-%s", p.v.doc.classLabel(), p.v.doc.mainSkillLabel())
		nameSeq[base]++
		if n := nameSeq[base]; n > 1 {
			base = fmt.Sprintf("%s-%d", base, n)
		}
		file := fmt.Sprintf("%04d-%s.xml", len(accepted)+1, base)
		if err := os.WriteFile(filepath.Join(outDir, file), p.x, 0o600); err != nil {
			return fmt.Errorf("cmdValidate: %w", err)
		}
		bySha[p.v.xmlSha1] = file
		accepted = append(accepted, Accepted{
			File: file, URL: p.l.URL, Host: p.l.Host, Source: p.l.Source,
			PostedAt: p.l.PostedAt, Context: p.l.Context,
			Class: p.v.doc.Build.ClassName, Ascend: p.v.doc.Build.AscendClassName,
			Skill: p.v.doc.mainSkillLabel(), Level: p.v.doc.Build.Level,
			Nodes: p.v.nodes, Slots: p.v.slots, Stats: p.v.stats, SHA1: p.v.xmlSha1,
		})
	}

	keep := make(map[string]bool, len(accepted))
	for _, a := range accepted {
		keep[a.File] = true
	}
	pruned, err := pruneStaleBuilds(outDir, keep)
	if err != nil {
		return err
	}
	if pruned > 0 {
		fmt.Printf("\npruned %d stale build(s) from a previous validate — %s now holds exactly the accepted set\n",
			pruned, outDir)
	}

	if err := writeJSONL(filepath.Join(*work, "builds.jsonl"), accepted); err != nil {
		return err
	}
	if err := writeJSONL(filepath.Join(*work, "rejects.jsonl"), rejects); err != nil {
		return err
	}

	total := len(links)
	fmt.Printf("\ncandidates      %d\n", total)
	fmt.Printf("ACCEPTED        %d unique builds  (%.1f%% of candidates)\n", len(accepted), pct(len(accepted), total))

	gate := len(accepted) + reasons[rejNoPlayerStat]
	fmt.Printf("  GATE tier        %6d  (tree + passives + slots; <PlayerStat> NOT required)\n", gate)
	fmt.Printf("  CROSS-CHECK tier %6d  (%.1f%% of gate-tier — these carry <PlayerStat>)\n",
		len(accepted), pct(len(accepted), gate))

	fmt.Println("\nrejection breakdown:")
	for _, r := range sortMap(reasons) {
		fmt.Printf("  %-22s %6d  %5.1f%%\n", r.Key, r.N, pct(r.N, total))
	}
	if len(wrongVer) > 0 {
		fmt.Println("\n  wrong_active_tree, by the version actually found:")
		for _, r := range sortMap(wrongVer) {
			fmt.Printf("    %-20s %6d\n", r.Key, r.N)
		}
	}
	if reasons[rejUnreachable] > 0 {
		fmt.Printf("\n⚠️  %d candidates were UNREACHABLE, not rejected. They are not evidence about the source.\n",
			reasons[rejUnreachable])
		fmt.Println("   Re-run `fetch` before treating this pass rate as final.")
	}

	fmt.Println("\nby source:")
	for _, r := range countByAccepted(accepted, func(a Accepted) string { return a.Source }) {
		fmt.Printf("  %-24s %6d\n", r.Key, r.N)
	}
	fmt.Println("by host:")
	for _, r := range countByAccepted(accepted, func(a Accepted) string { return a.Host }) {
		fmt.Printf("  %-24s %6d\n", r.Key, r.N)
	}
	fmt.Printf("\nbuilds -> %s\nprovenance -> %s\nrejects -> %s\n",
		outDir, filepath.Join(*work, "builds.jsonl"), filepath.Join(*work, "rejects.jsonl"))
	return nil
}

func validateExistingCorpus(dir string, c criteria, listRejects bool) error {
	banner("validate -corpus %s: tree=%s minNodes=%d minSlots=%d (%s)",
		dir, c.treeVersion, c.minNodes, c.minSlots, c.slotMode)

	entries, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return fmt.Errorf("validateExistingCorpus: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no *.xml under %s — refusing to report 0 rejections from an empty directory", dir)
	}

	reasons := map[string]int{}
	var failures []string
	var rejected []string
	ok := 0
	for _, f := range entries {
		x, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("validateExistingCorpus: %w", err)
		}
		v := judge(x, c)
		if v.ok {
			ok++
			continue
		}
		reasons[v.reason]++
		rejected = append(rejected, f)
		if len(failures) < 15 {
			failures = append(failures, fmt.Sprintf("  %s: %s %s (tree=%s nodes=%d slots=%d stats=%d)",
				filepath.Base(f), v.reason, v.detail, v.treeSeen, v.nodes, v.slots, v.stats))
		}
	}

	fmt.Printf("\nfiles           %d\nACCEPTED        %d (%.2f%%)\n", len(entries), ok, pct(ok, len(entries)))
	if len(reasons) == 0 {
		fmt.Println("rejections      none")
		return nil
	}
	fmt.Println("rejections:")
	for _, r := range sortMap(reasons) {
		fmt.Printf("  %-22s %6d\n", r.Key, r.N)
	}
	fmt.Println("first failures:")
	for _, l := range failures {
		fmt.Println(l)
	}
	if listRejects {
		fmt.Println("rejected paths:")
		for _, p := range rejected {
			fmt.Println(p)
		}
	}
	return fmt.Errorf("%d of %d files in %s failed the criteria", len(entries)-ok, len(entries), dir)
}

func sortLinksForNumbering(links []Link) {
	sort.Slice(links, func(i, j int) bool {
		ai, aj := links[i].PostedAt, links[j].PostedAt
		if (ai == "") != (aj == "") {
			return aj == ""
		}
		if ai != aj {
			return ai < aj
		}
		return links[i].URL < links[j].URL
	})
}

func orUnknown(s string) string {
	if s == "" {
		return "(no treeVersion attribute)"
	}
	return s
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

func sortMap(m map[string]int) []struct {
	Key string
	N   int
} {
	out := make([]struct {
		Key string
		N   int
	}, 0, len(m))
	for k, v := range m {
		out = append(out, struct {
			Key string
			N   int
		}{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func countByAccepted(as []Accepted, key func(Accepted) string) []struct {
	Key string
	N   int
} {
	m := map[string]int{}
	for _, a := range as {
		m[key(a)]++
	}
	return sortMap(m)
}

func writeJSONL[T any](path string, rows []T) error {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("writeJSONL: %w", err)
		}
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func pruneStaleBuilds(dir string, keep map[string]bool) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", dir, err)
	}
	removed := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".xml") || keep[name] {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return removed, fmt.Errorf("removing stale %s: %w", name, err)
		}
		removed++
	}
	return removed, nil
}
