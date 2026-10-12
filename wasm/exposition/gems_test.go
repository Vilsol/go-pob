package exposition

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestGetSkillGemsIsPopulatedAndClassified covers the gem list the frontend renders.
func TestGetSkillGemsIsPopulatedAndClassified(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	skillGemCache = nil

	gems := GetSkillGems()
	if len(gems) == 0 {
		t.Fatal("GetSkillGems returned an EMPTY LIST; the frontend gem picker has nothing to show")
	}
	if len(gems) < 500 {
		t.Errorf("GetSkillGems returned %d gems; PoB ships 608, so the filter is dropping most "+
			"of them", len(gems))
	}

	byID := make(map[string]SkillGem, len(gems))
	for _, g := range gems {
		byID[g.ID] = g
	}

	for _, c := range []struct {
		id   string
		want GemType
		name string
	}{
		{"Metadata/Items/Gems/SkillGemFireball", GemTypeIntelligence, "Fireball"},
		{"Metadata/Items/Gems/SkillGemHeavyStrike", GemTypeStrength, "Heavy Strike"},
		{"Metadata/Items/Gems/SkillGemDoubleStrike", GemTypeDexterity, "Double Strike"},
	} {
		g, ok := byID[c.id]
		if !ok {
			t.Errorf("%s is missing from the gem list", c.name)
			continue
		}
		if g.GemType != c.want {
			t.Errorf("%s classified as GemType %v, want %v", c.name, g.GemType, c.want)
		}
		if g.Base.Name != c.name {
			t.Errorf("%s has base name %q", c.name, g.Base.Name)
		}
		if g.Base.Description == "" {
			t.Errorf("%s has an empty description", c.name)
		}
		if g.MaxLevel <= 1 {
			t.Errorf("%s reports MaxLevel %d; its granted effect is not resolving (the zeroed FK "+
				"returns Fireball's for every gem)", c.name, g.MaxLevel)
		}
	}

	sup, ok := byID["Metadata/Items/Gems/SupportGemAddedFireDamage"]
	if !ok {
		t.Fatal("Added Fire Damage Support is missing from the gem list")
	}
	if !sup.Support {
		t.Error("Added Fire Damage Support is not flagged Support; the granted effect is resolving " +
			"to some other gem's")
	}
	if fb, ok := byID["Metadata/Items/Gems/SkillGemFireball"]; ok && fb.Support {
		t.Error("Fireball is flagged Support; every gem is resolving to the same granted effect")
	}
}

// TestVaalGemNonVaalHalfResolvesPerGem covers the Vaal arm of GetSkillGems, which the test above never reaches.
func TestVaalGemNonVaalHalfResolvesPerGem(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	skillGemCache = nil

	byID := make(map[string]SkillGem)
	for _, g := range GetSkillGems() {
		byID[g.ID] = g
	}

	fireball, ok := byID["Metadata/Items/Gems/SkillGemFireball"]
	if !ok {
		t.Fatal("Fireball is missing from the gem list")
	}
	doubleStrike, ok := byID["Metadata/Items/Gems/SkillGemDoubleStrike"]
	if !ok {
		t.Fatal("Double Strike is missing from the gem list")
	}

	vaalDoubleStrike, ok := byID["Metadata/Items/Gems/SkillGemVaalDoubleStrike"]
	if !ok {
		t.Fatal("Vaal Double Strike is missing from the gem list")
	}

	if vaalDoubleStrike.Vaal.Name != "Double Strike" {
		t.Errorf("Vaal Double Strike's non-Vaal half is named %q, want %q",
			vaalDoubleStrike.Vaal.Name, "Double Strike")
	}
	if got := vaalDoubleStrike.Vaal.Description; got != doubleStrike.Base.Description {
		t.Errorf("Vaal Double Strike's non-Vaal description is %q, want Double Strike's %q",
			got, doubleStrike.Base.Description)
	}
	if vaalDoubleStrike.Vaal.Description == fireball.Base.Description {
		t.Error("Vaal Double Strike's non-Vaal description is FIREBALL's — SkillGem.GetGrantedEffect() " +
			"reads a FK that is 0 on all 847 gems and returns GrantedEffects[0] for every one of them")
	}

	vaalFireball, ok := byID["Metadata/Items/Gems/SkillGemVaalFireball"]
	if !ok {
		t.Fatal("Vaal Fireball is missing from the gem list")
	}
	if vaalFireball.Vaal.Name != "Fireball" || vaalFireball.Vaal.Description != fireball.Base.Description {
		t.Errorf("Vaal Fireball's non-Vaal half is %q/%q, want Fireball's",
			vaalFireball.Vaal.Name, vaalFireball.Vaal.Description)
	}

	if fireball.Vaal != (GemPart{}) {
		t.Errorf("plain Fireball carries a non-empty Vaal part %+v", fireball.Vaal)
	}
}

// TestSkillGemsCarryPoBNameTagsAndLevelRequirement: GemSelectControl.lua:154-253 searches gemData.name.
func TestSkillGemsCarryPoBNameTagsAndLevelRequirement(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	byID := map[string]SkillGem{}
	for _, g := range GetSkillGems() {
		byID[g.ID] = g
	}
	fireball := byID["Metadata/Items/Gems/SkillGemFireball"]
	if fireball.Name != "Fireball" || !slices.Equal(fireball.Tags, []string{"area", "fire", "grants_active_skill", "intelligence", "projectile", "spell"}) || fireball.LevelRequirement != 1 {
		t.Errorf("Fireball: name %q tags %v level %d", fireball.Name, fireball.Tags, fireball.LevelRequirement)
	}
	for _, g := range GetSkillGems() {
		if g.Base.Name == "Trap Support" {
			if g.Name != "Trap" || !slices.Contains(g.Tags, "support") || g.LevelRequirement != 8 {
				t.Errorf("Trap Support: name %q tags %v level %d", g.Name, g.Tags, g.LevelRequirement)
			}
		}
	}
}

// TestGemErrorIsFindSkillGemsMessage: SkillsTab.lua:1137-1181: a gem named only by nameSpec (a pre-1.4.20 build) carries FindSkillGem's message.
func TestGemErrorIsFindSkillGemsMessage(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	if got := GemError("", "", "Fireball"); got != "" {
		t.Errorf("Fireball by name: %q", got)
	}
	if got := GemError("", "", "Zzzqx"); got != "Unrecognised gem name 'Zzzqx'" {
		t.Errorf("unknown name: %q", got)
	}
	if got := GemError("", "", "Fire"); !strings.HasPrefix(got, "Ambiguous gem name 'Fire': matches '") {
		t.Errorf("ambiguous name: %q", got)
	}
	if got := GemError("Metadata/Items/Gems/SkillGemFireball", "", "Zzzqx"); got != "" {
		t.Errorf("by id: %q", got)
	}
}

// TestVaalGemsNameTheirGlobalEffects: SkillsTab.lua:1014-1050: a Vaal gem shows an "Enable <name>:" box per granted effect that is no support.
func TestVaalGemsNameTheirGlobalEffects(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	byID := map[string]SkillGem{}
	for _, g := range GetSkillGems() {
		byID[g.ID] = g
	}
	if got := byID["Metadata/Items/Gems/SkillGemVaalFireball"].GlobalEffects; !slices.Equal(got, []string{"Vaal Fireball", "Fireball"}) {
		t.Errorf("Vaal Fireball: %q", got)
	}
	if got := byID["Metadata/Items/Gems/SkillGemFireball"].GlobalEffects; got != nil {
		t.Errorf("Fireball: %q", got)
	}
}
