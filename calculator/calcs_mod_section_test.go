package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

type modSectionCase struct {
	File      string   `json:"file"`
	ModName   []string `json:"modName"`
	Multi     bool     `json:"multi"`
	ModType   string   `json:"modType"`
	Cfg       string   `json:"cfg"`
	ModSource string   `json:"modSource"`
	Enemy     bool     `json:"enemy"`
	Actor     string   `json:"actor"`
	Rows      []struct {
		Value      string   `json:"value"`
		Name       string   `json:"name"`
		Flags      string   `json:"flags"`
		Tags       string   `json:"tags"`
		Source     string   `json:"source"`
		SourceName string   `json:"sourceName"`
		Totals     []string `json:"totals"`
	} `json:"rows"`
}

func modSectionRowText(value, name, flags, tags, source, sourceName string, totals []string) string {
	return strings.Join([]string{value, name, flags, tags, source, sourceName, strings.Join(totals, "/")}, " | ")
}

// TestCalcsModSectionAgainstPoB: Produced by oracle/calcs_mod_section.lua from the pinned PoB.
func TestCalcsModSectionAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/oracle/calcs_mod_section_expected.jsonl.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(zr)
	scanner.Buffer(make([]byte, 1<<22), 1<<22)
	envs := map[string]*Environment{}
	total, wrong := 0, 0
	for scanner.Scan() {
		var c modSectionCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		env := envs[c.File]
		if env == nil {
			raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), c.File))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(raw)
			if err != nil {
				t.Fatal(err)
			}
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeCalcs, NewEnvironmentCache())
			envs[c.File] = env
		}
		want := make([]string, 0, len(c.Rows))
		for _, r := range c.Rows {
			want = append(want, modSectionRowText(r.Value, r.Name, r.Flags, r.Tags, r.Source, r.SourceName, r.Totals))
		}
		rows := env.CalcsModSection(ModSectionData{ModName: c.ModName, Multi: c.Multi, ModType: c.ModType, Cfg: c.Cfg, ModSource: c.ModSource, Enemy: c.Enemy, Actor: c.Actor}, false)
		got := make([]string, 0, len(rows))
		for _, r := range rows {
			got = append(got, modSectionRowText(r.Value, r.Name, r.Flags, r.Tags, r.Source, r.SourceName, r.Totals))
		}
		slices.Sort(want)
		slices.Sort(got)
		total++
		if !slices.Equal(got, want) {
			wrong++
			if os.Getenv("CMS_ALL") != "" {
				t.Errorf("%s %v %s cfg=%s enemy=%v:\n got %q\nPoB %q", c.File, c.ModName, c.ModType, c.Cfg, c.Enemy, got, want)
			}
		}
	}
	if total < 1000 {
		t.Fatalf("only %d tables", total)
	}
	const ceiling = 3
	if wrong > ceiling {
		t.Errorf("%d of %d tables differ, ceiling %d", wrong, total, ceiling)
	} else if wrong < ceiling {
		t.Errorf("%d of %d tables differ: lower the ceiling from %d", wrong, total, ceiling)
	}
}

// TestCalcsModSectionItemRowsNameTheirItem: An Item source's row names its item.
func TestCalcsModSectionItemRowsNameTheirItem(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0001-necromancer-unknown.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeCalcs, NewEnvironmentCache())
	seen := 0
	for _, r := range env.CalcsModSection(ModSectionData{ModName: []string{"Life", "Mana", "EnergyShield", "Damage"}, Multi: true}, false) {
		if r.Source != "Item" {
			continue
		}
		item := ItemByName(build, r.ItemName)
		if item == nil || rarityColor[item.Rarity]+item.Name != r.SourceName {
			t.Errorf("row %q names %q, found %v", r.SourceName, r.ItemName, item)
		}
		seen++
	}
	if seen < 3 {
		t.Fatalf("only %d item rows: no control", seen)
	}
}

// TestCalcsModSectionTreeRowsCarryTheirNode: A Tree source's row names its node and carries its id.
func TestCalcsModSectionTreeRowsCarryTheirNode(t *testing.T) {
	build := parseCorpusBuild(t, "0021-assassin-blade-flurry.xml")
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeCalcs, NewEnvironmentCache())
	seen, cluster := 0, 0
	for _, r := range env.CalcsModSection(ModSectionData{ModName: []string{"Life", "Damage", "CritChance", "Speed", "PhysicalDamage"}, Multi: true}, false) {
		if r.Source != "Tree" {
			continue
		}
		seen++
		if r.NodeID == 0 || r.SourceName == "" {
			t.Errorf("tree row %q: node %d", r.SourceName, r.NodeID)
			continue
		}
		if isClusterNodeID(nodeKey(r.NodeID)) {
			cluster++
			if n, ok := env.clusterNodes[nodeKey(r.NodeID)]; !ok || n.Name == nil || *n.Name != r.SourceName {
				t.Errorf("cluster row %q: node %d", r.SourceName, r.NodeID)
			}
		}
	}
	if seen < 3 || cluster == 0 {
		t.Fatalf("%d tree rows, %d from cluster nodes: no control", seen, cluster)
	}
}
