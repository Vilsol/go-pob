package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func cmdManifest(args []string) error {
	fs := flag.NewFlagSet("manifest", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory for all stage artifacts")
	out := fs.String("out", "", "path to write (default <work>/builds/MANIFEST.md)")
	league := fs.String("league", "", "league name, e.g. \"Lake of Kalandra (3.19)\" (required)")
	tree := fs.String("tree", "", "tree version the corpus is pinned to, e.g. 3_19 (required)")
	from := fs.String("from", "", "window start YYYY-MM-DD (required)")
	to := fs.String("to", "", "window end YYYY-MM-DD (required)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("cmdManifest: %w", err)
	}
	for name, v := range map[string]string{"-league": *league, "-tree": *tree, "-from": *from, "-to": *to} {
		if v == "" {
			return fmt.Errorf("%s is required", name)
		}
	}

	accepted, err := readJSONL[Accepted](filepath.Join(*work, "builds.jsonl"))
	if err != nil {
		return err
	}
	if len(accepted) == 0 {
		return fmt.Errorf("no accepted builds in %s — run `validate` first", filepath.Join(*work, "builds.jsonl"))
	}
	rejects, err := readJSONL[Reject](filepath.Join(*work, "rejects.jsonl"))
	if err != nil {
		return err
	}
	fetched, err := loadFetchLog(*work)
	if err != nil {
		return err
	}

	path := *out
	if path == "" {
		path = filepath.Join(*work, "builds", "MANIFEST.md")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s Path of Building builds — verified\n\n", *league)
	fmt.Fprintf(&b, "**%d builds.** Every file was fetched as a PoB export code, inflated locally, and checked\n", len(accepted))
	fmt.Fprint(&b, "against the bytes actually written here — no source's claim about its own patch was trusted.\n\n")
	fmt.Fprint(&b, "Each file satisfies all three requirements:\n\n")
	fmt.Fprintf(&b, "1. the **active** `<Spec>` (per `<Tree activeSpec=\"N\">`) has `treeVersion=\"%s\"`. Many builds carry\n", *tree)
	fmt.Fprint(&b, "   several `<Spec>` entries at different versions (a stale leveling tree is common); only the active one counts;\n")
	fmt.Fprint(&b, "2. a `<PlayerStat>` list is present under `<Build>` (PoB's own computed output — the oracle);\n")
	fmt.Fprint(&b, "3. the character is real and fully equipped — >=80 allocated passives and >=8 filled item slots.\n\n")
	fmt.Fprint(&b, "Deduplicated by SHA-1 of the decompressed XML: no two files share content.\n\n")

	fmt.Fprint(&b, "## Provenance\n\n")
	candidates := len(accepted) + len(rejects)
	unreachable := 0
	for _, r := range rejects {
		if r.Reason == rejUnreachable {
			unreachable++
		}
	}
	fmt.Fprintf(&b, "Candidate PoB codes were gathered from the %s window (%s to %s) and every one was\n", *league, *from, *to)
	fmt.Fprintf(&b, "decoded locally before it was judged. %d codes were indexed in total: %d passed\n", candidates, len(accepted))
	fmt.Fprintf(&b, "(after content-hash dedup), %d were rejected on content, and %d were unreachable.\n\n",
		len(rejects)-unreachable, unreachable)

	fmt.Fprint(&b, "**Where the builds came from.** By host: ")
	fmt.Fprintln(&b, joinCounts(countByAccepted(accepted, func(a Accepted) string { return a.Host }))+".")
	fmt.Fprint(&b, "By source: ")
	fmt.Fprintln(&b, joinCounts(countByAccepted(accepted, func(a Accepted) string { return a.Source }))+".")
	fmt.Fprint(&b, "\n"+provenanceNarrative(accepted))

	viaProxy, direct := 0, 0
	for _, r := range fetched {
		if r.Class != fetchOK {
			continue
		}
		if r.Via == "proxy" {
			viaProxy++
		} else {
			direct++
		}
	}
	fmt.Fprint(&b, "**Transport.** ")
	fmt.Fprintf(&b, "%d codes were retrieved directly and %d had to be recovered through the `r.jina.ai` read proxy\n", direct, viaProxy)
	fmt.Fprint(&b, "after the origin served a Cloudflare interstitial. Transport failures are recorded as their own\n")
	fmt.Fprint(&b, "class and are never counted as content rejections — a block that masquerades as a rejection is how\n")
	fmt.Fprint(&b, "the 3.18 harvest nearly lost half its corpus.\n\n")

	fmt.Fprint(&b, "**Why candidates were rejected.**\n\n")
	fmt.Fprint(&b, "| reason | count |\n|---|---|\n")
	rc := map[string]int{}
	for _, r := range rejects {
		rc[r.Reason]++
	}
	for _, r := range sortMap(rc) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", r.Key, r.N)
	}
	fmt.Fprintln(&b)

	fmt.Fprint(&b, "## Coverage\n\n| Ascendancy | Builds | Distinct main skills |\n|---|---|---|\n")
	byAsc := map[string]map[string]bool{}
	for _, a := range accepted {
		k := a.Ascend
		if k == "" || k == "None" {
			k = a.Class + " (unascended)"
		}
		if byAsc[k] == nil {
			byAsc[k] = map[string]bool{}
		}
		byAsc[k][a.Skill] = true
	}
	ascKeys := make([]string, 0, len(byAsc))
	for k := range byAsc {
		ascKeys = append(ascKeys, k)
	}
	sort.Slice(ascKeys, func(i, j int) bool { return ascKeys[i] < ascKeys[j] })
	for _, k := range ascKeys {
		n := 0
		for _, a := range accepted {
			ak := a.Ascend
			if ak == "" || ak == "None" {
				ak = a.Class + " (unascended)"
			}
			if ak == k {
				n++
			}
		}
		fmt.Fprintf(&b, "| %s | %d | %d |\n", k, n, len(byAsc[k]))
	}
	fmt.Fprintln(&b)

	fmt.Fprint(&b, "## Per-build detail\n\n")
	for i, a := range accepted {
		fmt.Fprintf(&b, "### %d. `%s`\n\n", i+1, a.File)
		fmt.Fprintf(&b, "- source: %s (%s)\n", a.URL, a.Source)
		if a.Context != "" {
			fmt.Fprintf(&b, "- found in: %s\n", a.Context)
		}
		fmt.Fprintf(&b, "- posted: %s\n", a.PostedAt)
		fmt.Fprintf(&b, "- class: %s / %s, level %d\n", a.Class, a.Ascend, a.Level)
		fmt.Fprintf(&b, "- main skill: %s\n", a.Skill)
		fmt.Fprintf(&b, "- %d allocated passives, %d filled equipment slots, %d PlayerStat entries\n", a.Nodes, a.Slots, a.Stats)
		fmt.Fprintf(&b, "- sha1: `%s`\n\n", a.SHA1)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cmdManifest: %w", err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("cmdManifest: %w", err)
	}
	banner("manifest: %d builds -> %s", len(accepted), path)
	return nil
}

func provenanceNarrative(accepted []Accepted) string {
	var reddit, forum, poebuilds, pbAuthors, pobarchives bool
	for _, a := range accepted {
		switch {
		case strings.HasPrefix(a.Source, "r/"):
			reddit = true
		case a.Source == "poe-forum":
			forum = true
		case a.Source == "poebuilds.cc":
			poebuilds = true
		case a.Source == "poebuilds.cc-author":
			pbAuthors = true
		case a.Source == "pobarchives.com":
			pobarchives = true
		}
	}

	var out []string
	if reddit {
		out = append(out, "The Reddit material was pulled from the **arctic-shift** archive "+
			"(`arctic-shift.photon-reddit.com`),\nsweeping posts *and* comments across the whole window.")
	}
	if forum {
		out = append(out, "The forum material came from the class\nbuild listings plus every reply "+
			"page of the version-tagged build threads found there.")
	}
	if poebuilds {
		out = append(out, "The poebuilds.cc material came from that site's own\nper-league "+
			"ascendancy listings, which are indexed by league rather than by date and so carry no window.")
	}
	if pbAuthors {
		out = append(out, "The poebuilds.cc-author material came from that site's\n~267 sitemap-listed "+
			"author pages, which carry NO league filter at all and so span every league the site holds.")
	}
	if pobarchives {
		out = append(out, "The pobarchives.com material came from that site's own\nbuild directory, "+
			"paged at `?tree=<version>` — the only source here selected on the TREE VERSION, which is "+
			"criterion 1 itself, rather than on when a guide was written.")
	}
	if len(out) == 0 {
		return "⚠️ No source could be attributed to any accepted build. This manifest cannot say " +
			"where its corpus came from.\n\n"
	}
	return strings.Join(out, " ") + "\n\n"
}

func joinCounts(rows []struct {
	Key string
	N   int
}) string {
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%s %d", r.Key, r.N))
	}
	return strings.Join(parts, ", ")
}

func readJSONL[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("readJSONL: %w", err)
	}
	defer f.Close()

	var out []T
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for ln := 1; sc.Scan(); ln++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var v T
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, ln, err)
		}
		out = append(out, v)
	}
	return out, wrapScan(sc.Err())
}

func cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory for all stage artifacts")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	banner("corpus status: %s", *work)

	idx, err := loadIndex(*work)
	if err != nil {
		return err
	}
	report := func(stage, detail string, ran bool) {
		state := "NOT-RUN"
		if ran {
			state = "ok"
		}
		fmt.Printf("  %-10s %-8s %s\n", stage, state, detail)
	}

	report("discover", fmt.Sprintf("%d links indexed", len(idx.byID)), len(idx.byID) > 0)

	fetched, err := loadFetchLog(*work)
	if err != nil {
		return err
	}
	cls := map[fetchClass]int{}
	for _, r := range fetched {
		cls[r.Class]++
	}
	parts := make([]string, 0, len(cls))
	for _, k := range sortedClasses(cls) {
		parts = append(parts, fmt.Sprintf("%s=%d", k, cls[k]))
	}
	report("fetch", fmt.Sprintf("%d attempted [%s]", len(fetched), strings.Join(parts, " ")), len(fetched) > 0)

	accepted, err := readJSONL[Accepted](filepath.Join(*work, "builds.jsonl"))
	if err != nil {
		return err
	}
	rejects, err := readJSONL[Reject](filepath.Join(*work, "rejects.jsonl"))
	if err != nil {
		return err
	}
	report("validate", fmt.Sprintf("%d accepted / %d rejected", len(accepted), len(rejects)),
		len(accepted)+len(rejects) > 0)

	mpath := filepath.Join(*work, "builds", "MANIFEST.md")
	st, serr := os.Stat(mpath)
	d := "not written"
	if serr == nil {
		d = fmt.Sprintf("%d KB, %s", st.Size()/1024, st.ModTime().UTC().Format("2006-01-02 15:04"))
	}
	report("manifest", d, serr == nil)
	return nil
}
