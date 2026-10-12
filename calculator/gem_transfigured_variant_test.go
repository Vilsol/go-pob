package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestTransfiguredVariantRowsAreReachable pins the fix for a whole class of silently wrong gem keyword matches.
func TestTransfiguredVariantRowsAreReachable(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	base := poe.BaseItemTypeByIDMap["Metadata/Items/Gems/SkillGemLancingSteel"]
	if base == nil {
		t.Fatal("go-pob-data no longer ships Lancing Steel; pick another transfigured family")
	}
	gem := base.SkillGem()
	if gem == nil {
		t.Fatal("Lancing Steel's BaseItemType has no SkillGem")
	}

	if !CalcGemIsType(gem, "physical") {
		t.Fatal("control: the base Lancing Steel row lost its `physical` tag")
	}

	variant, ok := raw.GemVariants["LancingSteelAltX"]
	if !ok {
		t.Fatal("raw.GemVariants has no LancingSteelAltX: the shadowed rows are being dropped again")
	}
	if variant.Name != "Lancing Steel of Spraying" {
		t.Fatalf("LancingSteelAltX resolved to %q, not the variant row", variant.Name)
	}
	if variant.Tags["physical"] {
		t.Fatal("the pinned LancingSteelAltX row declares no `physical` tag; the fixture is stale")
	}

	if CalcGemIsTypeVariant(gem, "LancingSteelAltX", "physical") {
		t.Error("CalcGemIsTypeVariant read the BASE row's tags for a transfigured instance")
	}
	if !CalcGemIsTypeVariant(gem, "LancingSteelAltX", "projectile") {
		t.Error("CalcGemIsTypeVariant lost a tag the variant row does declare")
	}
	if !CalcGemIsTypeVariant(gem, "LancingSteelAltX", "all") {
		t.Error(`the variant path lost gemIsType's "all" arm`)
	}
	if !CalcGemIsTypeVariant(gem, "LancingSteelAltX", "All") {
		t.Error("the variant path lost the case fold")
	}
	if !CalcGemIsTypeVariant(gem, "LancingSteelAltX", "lancing steel") {
		t.Error("the transfigured variant no longer answers to its base gem's name")
	}
	if CalcGemIsType(gem, "lancing steel of spraying") {
		t.Error("CalcGemIsType gained the includeTransfigured arm ModStore.lua:737 does not pass")
	}
	if !CalcGemIsTypeVariant(gem, "LancingSteel", "physical") {
		t.Error("a non-transfigured granted effect no longer falls back to the base row")
	}
	if !CalcGemIsTypeVariant(gem, "", "physical") {
		t.Error("an empty granted effect no longer falls back to the base row")
	}
}

// TestGemVariantsCannotShadowABaseGem asserts the property that makes the variant table safe to consult FIRST.
func TestGemVariantsCannotShadowABaseGem(t *testing.T) {
	claimed := make(map[string]string, len(raw.Gems))
	for gameID, row := range raw.Gems {
		if row.GrantedEffectID != "" {
			claimed[row.GrantedEffectID] = gameID
		}
	}
	for effectID := range raw.GemVariants {
		if gameID, ok := claimed[effectID]; ok {
			t.Errorf("GemVariants[%q] shadows the base gem %s, which claims the same granted effect",
				effectID, gameID)
		}
		if _, ok := raw.Gems[effectID]; ok {
			t.Errorf("GemVariants[%q] collides with a raw.Gems key", effectID)
		}
	}
	if len(raw.GemVariants) == 0 {
		t.Fatal("GemVariants is empty: this test would pass vacuously")
	}
}
