package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	poebuildsHost = "www.poebuilds.cc"
	poebuildsBase = "https://" + poebuildsHost
)

var (
	rePBSection   = regexp.MustCompile(`href="/poe1/([a-z][a-z-]*)/([a-z][a-z-]*)"`)
	rePBLeagueOpt = regexp.MustCompile(`<option value="([a-z0-9-]+)" data-version="([0-9.]+)"( selected)?>`)
)

type pbLeague struct {
	slug     string
	version  string
	selected bool
}

func parsePBLeagues(body []byte) []pbLeague {
	ms := rePBLeagueOpt.FindAllStringSubmatch(string(body), -1)
	out := make([]pbLeague, 0, len(ms))
	seen := map[string]bool{}
	for _, m := range ms {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, pbLeague{slug: m[1], version: m[2], selected: m[3] != ""})
	}
	return out
}

func parsePBSections(body []byte) []string {
	ms := rePBSection.FindAllStringSubmatch(string(body), -1)
	out := make([]string, 0, len(ms))
	seen := map[string]bool{}
	for _, m := range ms {
		class, asc := m[1], m[2]
		if class == "author" {
			continue
		}
		path := "/poe1/" + class + "/" + asc
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

const minPBSections = 15

func discoverPoebuilds(ctx context.Context, c *client, idx *linkIndex, st *discoverStats, wantLeague string, verbose bool) error {
	body, err := getWithRetry(ctx, c, poebuildsHost, poebuildsBase+"/poe1", 4, verbose, true)
	st.requests++
	if err != nil {
		return fmt.Errorf("poebuilds.cc index: %w", err)
	}

	leagues := parsePBLeagues(body)
	if len(leagues) == 0 {
		return fmt.Errorf("poebuilds.cc: no league selector on /poe1 — the markup changed; " +
			"refusing rather than crawling an unidentified league")
	}

	slug := ""
	if wantLeague != "" {
		for _, l := range leagues {
			if l.slug == wantLeague || l.version == wantLeague {
				slug = l.slug
				break
			}
		}
		if slug == "" {
			return fmt.Errorf("poebuilds.cc: -pb-league %q is neither a slug nor a version the site offers.\n%s",
				wantLeague, pbLeagueTable(leagues))
		}
	}

	sections := parsePBSections(body)
	if len(sections) < minPBSections {
		return fmt.Errorf("poebuilds.cc: only %d class/ascendancy sections parsed from /poe1 (want >= %d) — "+
			"the markup changed; refusing rather than reporting a thin source", len(sections), minPBSections)
	}

	label := "site default (current league)"
	if slug != "" {
		label = slug
	}
	fmt.Printf("  poebuilds.cc  %d sections, league=%s\n", len(sections), label)

	before := len(idx.byID)
	served := map[string]int{}
	for _, path := range sections {
		u := poebuildsBase + path
		if slug != "" {
			u += "?league=" + slug
		}
		pageBody, err := getWithRetry(ctx, c, poebuildsHost, u, 4, verbose, true)
		st.requests++
		if err != nil {
			return fmt.Errorf("poebuilds.cc %s: %w", path, err)
		}
		for _, l := range parsePBLeagues(pageBody) {
			if l.selected {
				served[l.version+" ("+l.slug+")"]++
			}
		}
		st.harvest(idx, string(pageBody), "poebuilds.cc", poebuildsBase+path, 0)
		st.flush(false)
	}

	fmt.Printf("  poebuilds.cc  %d new links from %d sections\n", len(idx.byID)-before, len(sections))
	fmt.Print("  served league: ")
	if len(served) == 0 {
		fmt.Print("NONE — no page carried a selected league")
	}
	for k, n := range served {
		fmt.Printf("%s x%d  ", k, n)
	}
	fmt.Println()

	if slug != "" {
		for k := range served {
			if !strings.Contains(k, "("+slug+")") {
				return fmt.Errorf("poebuilds.cc: asked for league %q but a page served %s — "+
					"the links just indexed are the WRONG league", slug, k)
			}
		}
	}
	return nil
}

var rePBAuthor = regexp.MustCompile(`<loc>(https://[a-z.]*poebuilds\.cc/poe1/author/[a-z0-9-]+)</loc>`)

const minPBAuthors = 50

func parsePBAuthors(body []byte) []string {
	ms := rePBAuthor.FindAllStringSubmatch(string(body), -1)
	out := make([]string, 0, len(ms))
	seen := map[string]bool{}
	for _, m := range ms {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

func discoverPoebuildsAuthors(ctx context.Context, c *client, idx *linkIndex, st *discoverStats, verbose bool) error {
	body, err := getWithRetry(ctx, c, poebuildsHost, poebuildsBase+"/sitemap.xml", 4, verbose, true)
	st.requests++
	if err != nil {
		return fmt.Errorf("poebuilds.cc sitemap: %w", err)
	}
	authors := parsePBAuthors(body)
	if len(authors) < minPBAuthors {
		return fmt.Errorf("poebuilds.cc: only %d author URLs parsed from the sitemap (want >= %d) — "+
			"the sitemap or its markup changed; refusing rather than reporting a thin source",
			len(authors), minPBAuthors)
	}
	fmt.Printf("  poebuilds.cc  %d author pages from the sitemap (NO league filter — every league at once)\n", len(authors))

	before := len(idx.byID)
	for _, u := range authors {
		pageBody, err := getWithRetry(ctx, c, poebuildsHost, u, 4, verbose, true)
		st.requests++
		if err != nil {
			return fmt.Errorf("poebuilds.cc %s: %w", u, err)
		}
		st.harvest(idx, string(pageBody), "poebuilds.cc-author", u, 0)
		st.flush(false)
	}
	fmt.Printf("  poebuilds.cc  %d new links from %d author pages\n", len(idx.byID)-before, len(authors))
	return nil
}

func pbLeagueTable(leagues []pbLeague) string {
	var b strings.Builder
	b.WriteString("  available (slug = version):\n")
	for _, l := range leagues {
		b.WriteString(fmt.Sprintf("    %-12s %s\n", l.slug, l.version))
	}
	return b.String()
}
