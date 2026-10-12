package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

// TestGemInstanceNameIsPoBGemName pins CalcPerform.lua:571-572.
func TestGemInstanceNameIsPoBGemName(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct{ gemID, skillID, want string }{
		{"Metadata/Items/Gems/SkillGemVulnerability", "Despair", "Despair"},
		{"Metadata/Items/Gems/SkillGemNewVulnerability", "Vulnerability", "Vulnerability"},
		{"Metadata/Items/Gems/SkillGemTemporalChains", "TemporalChains", "Temporal Chains"},
		{"Metadata/Items/Gems/SupportGemBlasphemy", "SupportBlasphemy", "Blasphemy"},
	} {
		got := gemInstanceName(pob.Gem{GemID: c.gemID, SkillID: c.skillID})
		if got != c.want {
			t.Errorf("gemInstanceName(%q) = %q, want %q — CalcPerform.lua:572 compares "+
				"gemData.name, and PoB's gem table is the only place that string lives",
				c.gemID, got, c.want)
		}
	}

	seen := map[string]string{}
	for _, gemID := range []string{
		"Metadata/Items/Gems/SkillGemVulnerability",
		"Metadata/Items/Gems/SkillGemNewVulnerability",
		"Metadata/Items/Gems/SkillGemTemporalChains",
	} {
		name := gemInstanceName(pob.Gem{GemID: gemID})
		if prior, dup := seen[name]; dup {
			t.Errorf("CONTROL: %q and %q both answer %q — the accessor is still returning a "+
				"single constant for every gem", prior, gemID, name)
		}
		seen[name] = gemID
	}

	if got := gemInstanceName(pob.Gem{SkillID: "Despair"}); got != "" {
		t.Errorf("gemInstanceName with no gemId = %q, want \"\" — SkillsTab.lua:998-1007 leaves "+
			"gemData nil for an item-granted skill and CalcPerform.lua:572 short-circuits on it", got)
	}
}

// TestCursePriorityReadsSocketIndex is the consumer half.
func TestCursePriorityReadsSocketIndex(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	support := pob.Gem{GemID: "Metadata/Items/Gems/SupportGemBlasphemy", SkillID: "SupportBlasphemy"}
	despair := pob.Gem{GemID: "Metadata/Items/Gems/SkillGemVulnerability", SkillID: "Despair"}

	priorityWithGems := func(gems ...pob.Gem) float64 {
		return determineCursePriority("Despair", &ActiveSkill{
			SocketGroup: &pob.Skill{Slot: "Helmet", Gems: gems},
		})
	}

	first := priorityWithGems(despair, support)
	second := priorityWithGems(support, despair)

	if first == second {
		t.Fatalf("Despair in socket 1 and socket 2 both give priority %v — CalcPerform.lua:574 "+
			"charges m_min(k, 8) * cursePriority[\"SocketPriorityBase\"] and the index is being lost",
			first)
	}

	step := cursePriority["SocketPriorityBase"]
	if got := second - first; got != step {
		t.Errorf("socket 2 minus socket 1 = %v, want %v (one SocketPriorityBase) — the index is "+
			"read but not charged per PoB's formula", got, step)
	}

	nine := make([]pob.Gem, 0, 9)
	for range 8 {
		nine = append(nine, support)
	}
	nine = append(nine, despair)
	eight := append(append([]pob.Gem{}, nine[:7]...), despair)

	if got, want := priorityWithGems(nine...), priorityWithGems(eight...); got != want {
		t.Errorf("Despair in socket 9 = %v, socket 8 = %v — m_min(k, 8) must clamp them equal", got, want)
	}
}
