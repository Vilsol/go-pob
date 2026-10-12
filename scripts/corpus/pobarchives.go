package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	pobarchivesHost = "pobarchives.com"
	pobarchivesBase = "https://" + pobarchivesHost
)

var rePABuild = regexp.MustCompile(`href="/build/([A-Za-z0-9_-]+)"`)

const minPAListingIDs = 5

const maxPAListingPages = 400

const paRenderBudget = 60 * time.Second

func paCheckpointKey(id string) string { return "pobarchives/build/" + id }

func parsePABuildIDs(body []byte) []string {
	ms := rePABuild.FindAllStringSubmatch(string(body), -1)
	out := make([]string, 0, len(ms))
	seen := map[string]bool{}
	for _, m := range ms {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, m[1])
	}
	return out
}

func paReadyListing(html string) bool { return rePABuild.MatchString(html) }
func paReadyBuild(html string) bool   { return strings.Contains(html, "pobb.in/") }

func discoverPobarchives(ctx context.Context, c *client, idx *linkIndex, st *discoverStats, ck *checkpoints, tree string, verbose bool, br *cdpBrowser) error {
	if tree == "" {
		return fmt.Errorf("pobarchives: -pa-tree is required (e.g. 3.29) — an unfiltered sweep is " +
			"2050 listing pages and 20496 build pages of every league at once")
	}
	tree = strings.ReplaceAll(tree, "_", ".")

	get := func(u string, ready func(string) bool) ([]byte, error) {
		if br == nil {
			return getWithRetry(ctx, c, pobarchivesHost, u, 4, verbose, true)
		}
		done := c.lim.wait(pobarchivesHost)
		defer done()
		html, _, err := br.render(ctx, u, ready, paRenderBudget)
		if err != nil {
			return nil, err
		}
		if verbose {
			fmt.Printf("      CDP %s -> %d bytes\n", u, len(html))
		}
		return []byte(html), nil
	}

	var ids []string
	for page := 1; ; page++ {
		if page > maxPAListingPages {
			return fmt.Errorf("pobarchives: listing exceeded %d pages — the paginator is not terminating", maxPAListingPages)
		}
		u := fmt.Sprintf("%s/builds/poe?tree=%s&goldenPage=%d", pobarchivesBase, tree, page)
		body, err := get(u, paReadyListing)
		st.requests++
		if err != nil {
			return fmt.Errorf("pobarchives listing page %d: %w", page, err)
		}
		got := parsePABuildIDs(body)
		if page == 1 && len(got) < minPAListingIDs {
			return fmt.Errorf("pobarchives: page 1 of tree=%s yielded %d build ids (want >= %d) — "+
				"either the markup changed or that tree has no builds, and this tool cannot tell "+
				"them apart, so it refuses rather than reporting a thin source", tree, len(got), minPAListingIDs)
		}
		if len(got) == 0 {
			break
		}
		ids = append(ids, got...)
	}
	uniq := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			uniq = append(uniq, id)
		}
	}
	fmt.Printf("  pobarchives   %d build pages at tree=%s (%d listing rows, %d duplicates)\n",
		len(uniq), tree, len(ids), len(ids)-len(uniq))

	before := len(idx.byID)
	empty, resumed := 0, 0
	for i, id := range uniq {
		if ck.complete(paCheckpointKey(id)) {
			resumed++
			continue
		}
		body, err := get(pobarchivesBase+"/build/"+id, paReadyBuild)
		st.requests++
		if err != nil {
			return fmt.Errorf("pobarchives build %s (%d of %d): %w", id, i+1, len(uniq), err)
		}
		n := len(idx.byID)
		st.harvest(idx, string(body), "pobarchives.com", pobarchivesBase+"/build/"+id, 0)
		if len(idx.byID) == n {
			empty++
		}
		ck.done(paCheckpointKey(id))
		st.flush(false)
	}
	fmt.Printf("  pobarchives   %d new links from %d build pages (%d pages carried nothing new, %d skipped as already crawled)\n",
		len(idx.byID)-before, len(uniq)-resumed, empty, resumed)
	return nil
}
