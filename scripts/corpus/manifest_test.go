package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProvenanceNarrativeNamesOnlyTheSourcesUsed pins the manifest's source prose to the accepted set.
func TestProvenanceNarrativeNamesOnlyTheSourcesUsed(t *testing.T) {
	const (
		redditSentence    = "arctic-shift"
		forumSentence     = "forum material"
		poebuildsSentence = "poebuilds.cc material"
	)

	cases := []struct {
		name    string
		sources []string
		want    []string
		absent  []string
	}{
		{
			name:    "reddit only",
			sources: []string{"r/PathOfExileBuilds", "r/pathofexile"},
			want:    []string{redditSentence},
			absent:  []string{forumSentence, poebuildsSentence},
		},
		{
			name:    "forum only",
			sources: []string{"poe-forum"},
			want:    []string{forumSentence},
			absent:  []string{redditSentence, poebuildsSentence},
		},
		{
			name:    "poebuilds only",
			sources: []string{"poebuilds.cc"},
			want:    []string{poebuildsSentence},
			absent:  []string{redditSentence, forumSentence},
		},
		{
			name:    "all three",
			sources: []string{"r/PathOfExileBuilds", "poe-forum", "poebuilds.cc"},
			want:    []string{redditSentence, forumSentence, poebuildsSentence},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			accepted := make([]Accepted, 0, len(tc.sources))
			for _, s := range tc.sources {
				accepted = append(accepted, Accepted{Source: s})
			}
			got := provenanceNarrative(accepted)

			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("narrative omits %q for sources %v.\ngot: %s", w, tc.sources, got)
				}
			}
			for _, a := range tc.absent {
				if strings.Contains(got, a) {
					t.Errorf("narrative claims %q for sources %v, which never contributed a build. "+
						"A manifest that describes a crawl it did not run is a false provenance "+
						"record.\ngot: %s", a, tc.sources, got)
				}
			}
		})
	}
}

// TestProvenanceNarrativeRefusesRatherThanRenderingNothing covers the case the repo keeps re-learning.
func TestProvenanceNarrativeRefusesRatherThanRenderingNothing(t *testing.T) {
	got := provenanceNarrative([]Accepted{{Source: "some-source-nobody-has-taught-this-yet"}})
	if !strings.Contains(got, "cannot say") {
		t.Errorf("an unrecognised source rendered silently instead of refusing.\ngot: %q", got)
	}
}

// TestPruneStaleBuildsLeavesExactlyTheAcceptedSet pins the fix for a defect that nearly poisoned the 3.29 corpus.
func TestPruneStaleBuildsLeavesExactlyTheAcceptedSet(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"1-keep.xml", "2-keep.xml", "9-stale.xml", "10-stale.xml"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("<x/>"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "MANIFEST.md"), []byte("# m"), 0o600); err != nil {
		t.Fatal(err)
	}

	keep := map[string]bool{"1-keep.xml": true, "2-keep.xml": true}
	removed, err := pruneStaleBuilds(dir, keep)
	if err != nil {
		t.Fatalf("pruneStaleBuilds: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed %d stale files, want 2", removed)
	}

	got := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		got[e.Name()] = true
	}
	for _, want := range []string{"1-keep.xml", "2-keep.xml", "MANIFEST.md"} {
		if !got[want] {
			t.Errorf("%s was deleted; only stale .xml may be removed", want)
		}
	}
	for _, gone := range []string{"9-stale.xml", "10-stale.xml"} {
		if got[gone] {
			t.Errorf("%s survived — a re-validate leaves a directory whose COUNT is wrong while "+
				"every file in it is real, which is how 2148 wrong-league builds nearly reached "+
				"the 3.29 corpus", gone)
		}
	}
}
