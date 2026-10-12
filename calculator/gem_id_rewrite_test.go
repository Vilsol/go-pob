package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestArcticArmourGemResolvesItsPoBRow pins the one entry of builds.pobGemIDtoGameGemIDs that pointed the WRONG WAY.
func TestArcticArmourGemResolvesItsPoBRow(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0006-elementalist-storm-burst.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, set := range build.Skills.SkillSets {
		for _, group := range set.Skills {
			for _, gem := range group.Gems {
				if gem.SkillID != "ArcticArmour" {
					continue
				}
				found = true
				gemData, _ := resolveGemInstance(gem)
				if gemData == nil {
					t.Fatalf("Arctic Armour (gemId %q) resolves to no SkillGem", gem.GemID)
				}
				pobGem, ok := pobGemFor(gemData)
				if !ok {
					base := "?"
					if b := gemData.GetBaseItemType(); b != nil {
						base = b.ID
					}
					t.Fatalf("Arctic Armour resolved to base %q, which has no raw.Gems row: "+
						"every gemIsType keyword answers false and its attribute requirement is 0", base)
				}
				if !pobGem.Tags["cold"] || !pobGem.Tags["spell"] || !pobGem.Tags["duration"] {
					t.Errorf("Arctic Armour tags = %v, want cold/spell/duration", pobGem.Tags)
				}
				if pobGem.ReqDex != 60 || pobGem.ReqInt != 40 || pobGem.ReqStr != 0 {
					t.Errorf("Arctic Armour reqs = str %d dex %d int %d, want 0/60/40",
						pobGem.ReqStr, pobGem.ReqDex, pobGem.ReqInt)
				}
				if !CalcGemIsType(gemData, "cold") {
					t.Error("CalcGemIsType(ArcticArmour, \"cold\") is false")
				}
			}
		}
	}
	if !found {
		t.Fatal("REFUSING: the fixture build carries no Arctic Armour gem — wrong fixture")
	}

	for from, to := range builds.PoBGemIDRewrites() {
		if poe.BaseItemTypeByIDMap[from] != nil {
			t.Errorf("rewrite %q -> %q: the SOURCE is a live BaseItemType, so the rewrite is "+
				"redirecting a valid game id", from, to)
		}
	}
}
