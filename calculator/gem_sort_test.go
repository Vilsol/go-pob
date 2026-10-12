package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

var gemSortKnownSkills = []string{"GeneralsCry", "SupportCrustaceousGrasp", "SupportAddedChaosDamage", "SupportIncreasedCriticalStrikes"}

var gemSortKnownGroups = map[string]bool{"0102-gladiator-raise-spectre.xml/1": true, "0023-elementalist-summon-ice-golem-of-hordes.xml/1": true}

// TestGemSortCacheAgainstPoB: Produced by oracle/gem_sort_cache.lua from the pinned PoB.
func TestGemSortCacheAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/oracle/gem_sort_cache_expected.jsonl.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		File       string      `json:"file"`
		Gems       [][3]string `json:"gems"`
		Group      int         `json:"group"`
		CanSupport []string    `json:"canSupport"`
		Pending    []string    `json:"pending"`
	}
	scanner := bufio.NewScanner(zr)
	scanner.Buffer(make([]byte, 1<<22), 1<<22)
	var env *Environment
	var gems []pob.Gem
	groups, wrong := 0, 0
	for scanner.Scan() {
		var r row
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if r.Gems != nil {
			env = sidebarEnv(t, r.File)
			gems = gems[:0]
			for _, g := range r.Gems {
				gem := pob.Gem{GemID: g[0], SkillID: g[1], Level: 20, Enabled: true}
				if g[2] != "" {
					v := g[2]
					gem.VariantID = &v
				}
				gems = append(gems, gem)
			}
			continue
		}
		groups++
		var canSupport, pending []string
		for i, s := range env.GemSortCache(r.Group, gems) {
			if s.CanSupport {
				canSupport = append(canSupport, gems[i].SkillID)
			}
			if s.Pending {
				pending = append(pending, gems[i].SkillID)
			}
		}
		known := func(list []string) []string {
			return slices.DeleteFunc(slices.Clone(list), func(id string) bool { return slices.Contains(gemSortKnownSkills, id) })
		}
		canSupport, pending, r.CanSupport, r.Pending = known(canSupport), known(pending), known(r.CanSupport), known(r.Pending)
		slices.Sort(canSupport)
		slices.Sort(pending)
		if gemSortKnownGroups[fmt.Sprintf("%s/%d", r.File, r.Group)] {
			if slices.Equal(canSupport, r.CanSupport) {
				t.Errorf("%s group %d now matches PoB: drop it from gemSortKnownGroups", r.File, r.Group)
			}
			continue
		}
		if !slices.Equal(canSupport, r.CanSupport) || !slices.Equal(pending, r.Pending) {
			wrong++
			if wrong <= 5 || os.Getenv("GSC_ALL") != "" {
				t.Errorf("%s group %d:\n canSupport got %d PoB %d, only ours %v, only PoB's %v\n pending got %d PoB %d, only ours %v, only PoB's %v", r.File, r.Group,
					len(canSupport), len(r.CanSupport), minus(canSupport, r.CanSupport), minus(r.CanSupport, canSupport), len(pending), len(r.Pending), minus(pending, r.Pending), minus(r.Pending, pending))
			}
		}
	}
	if groups < 100 {
		t.Fatalf("only %d groups", groups)
	}
	if wrong > 0 {
		t.Errorf("%d of %d groups differ", wrong, groups)
	}
}

func minus(a, b []string) []string {
	var out []string
	for _, s := range a {
		if !slices.Contains(b, s) {
			out = append(out, s)
		}
	}
	return out
}
