package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestEverySupportGemResolves asserts that every SUPPORT gem go-pob-data ships resolves in raw.Gems.
func TestEverySupportGemResolves(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	notInPoB := map[string]bool{
		"Metadata/Items/Gems/SupportGemCastLinkedCursesOnCurse": true,
		"Metadata/Items/Gems/SupportGemConflagration":           true,
		"Metadata/Items/Gems/SupportGemEdify":                   true,
		"Metadata/Items/Gems/SupportGemHandcastRapidFire":       true,
		"Metadata/Items/Gems/SupportGemHarrowingThrong":         true,
		"Metadata/Items/Gems/SupportGemMagnetism":               true,
		"Metadata/Items/Gems/SupportGemSplit":                   true,
		"Metadata/Items/Gems/SupportGemTemporaryForTutorial":    true,
		"Metadata/Items/Gems/SupportGemVaalSoulHarvesting":      true,
		"Metadata/Items/Gems/SupportGemWard":                    true,
	}

	var missing []string
	total := 0
	for _, gem := range poe.SkillGems {
		if gem == nil || gem.GetBaseItemType() == nil {
			continue
		}
		base := gem.GetBaseItemType()
		if strings.Contains(base.ID, "Royale") || !strings.Contains(base.ID, "SupportGem") ||
			notInPoB[base.ID] {
			continue
		}
		total++
		if _, ok := raw.Gems[base.ID]; !ok {
			missing = append(missing, base.ID)
		}
	}

	if total < 100 {
		t.Fatalf("only %d support gems enumerated — the probe is broken, not the table", total)
	}
	if len(missing) > 0 {
		show := missing
		if len(show) > 16 {
			show = show[:16]
		}
		t.Errorf("%d of %d SUPPORT gems do not resolve in raw.Gems, so they answer false to every "+
			"keyword — including \"+N to Level of Socketed Support Gems\": %v", len(missing), total, show)
	}
}
