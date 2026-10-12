package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestVaalTransfiguredSecondaryResolvesTheAlternate pins Data.lua:1098-1119's load-time synthesis.
func TestVaalTransfiguredSecondaryResolvesTheAlternate(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	base := poe.BaseItemTypeByIDMap["Metadata/Items/Gems/SkillGemVaalColdSnap"]
	if base == nil {
		t.Fatal("go-pob-data no longer ships Vaal Cold Snap; pick another vaal transfigured family")
	}
	gem := base.SkillGem()
	if gem == nil {
		t.Fatal("Vaal Cold Snap's BaseItemType has no SkillGem")
	}
	if !gem.IsVaalGem {
		t.Fatal("control: Vaal Cold Snap no longer reports IsVaalGem, so the synthesis arm is unreachable")
	}
	if poe.GrantedEffectByID("ColdSnapAltX") == nil {
		t.Fatal("ColdSnapAltX is absent from the loaded data; the fixture is stale")
	}

	altX := "VaalColdSnapAltX"
	altY := "VaalColdSnapAltY"
	unknown := "VaalColdSnap"

	for _, c := range []struct {
		name    string
		variant *string
		want    string
	}{
		{"absent variantId falls back to the base secondary", nil, "ColdSnap"},
		{"a variantId naming no alternate falls back too", &unknown, "ColdSnap"},
		{"AltX resolves the synthesised alternate", &altX, "ColdSnapAltX"},
		{"AltY resolves the synthesised alternate", &altY, "ColdSnapAltY"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := secondaryGrantedEffectOf(gem, "VaalColdSnap", c.variant)
			if got == nil {
				t.Fatalf("secondaryGrantedEffectOf = nil, want %q", c.want)
			}
			if got.ID != c.want {
				t.Errorf("secondaryGrantedEffectOf = %q, want %q", got.ID, c.want)
			}
		})
	}

	b := poe.BaseItemTypeByIDMap["Metadata/Items/Gems/SkillGemDarkRitual"]
	if b == nil || b.SkillGem() == nil {
		t.Fatal("go-pob-data no longer ships Bane (Metadata/Items/Gems/SkillGemDarkRitual); " +
			"pick another non-vaal family that declares its own variant secondary")
	}
	{
		bg := b.SkillGem()
		if bg.IsVaalGem {
			t.Fatal("control: Bane reports IsVaalGem, so it cannot separate the two arms")
		}
		baneAlt := "BaneAltX"
		for _, c := range []struct {
			name            string
			grantedEffectID string
			variant         *string
			want            string
		}{
			{"suffix alone does not synthesise for a non-vaal gem", "", &baneAlt, "SupportDarkRitual"},
			{"the declared variant row's own secondary wins", "BaneAltX", &baneAlt, "SupportDarkRitualAltX"},
			{"the base gem keeps the base secondary", "Bane", nil, "SupportDarkRitual"},
		} {
			t.Run(c.name, func(t *testing.T) {
				got := secondaryGrantedEffectOf(bg, c.grantedEffectID, c.variant)
				if got == nil {
					t.Fatalf("secondaryGrantedEffectOf = nil, want %q", c.want)
				}
				if got.ID != c.want {
					t.Errorf("secondaryGrantedEffectOf = %q, want %q", got.ID, c.want)
				}
			})
		}
	}
}

// TestVaalTransfiguredSecondaryArrivesOnACarrier is the ARRIVAL half, and it is the half that separates "not graded here" from "never lands".
func TestVaalTransfiguredSecondaryArrivesOnACarrier(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const name = "0076-occultist-vaal-cold-snap-of-power.xml"
	rawXML, err := os.ReadFile("../testdata/corpus-3.29/" + name)
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(rawXML)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	seen := map[string]bool{}
	for _, sk := range env.Player.ActiveSkillList {
		if sk.ActiveEffect != nil && sk.ActiveEffect.GrantedEffect != nil {
			seen[sk.ActiveEffect.GrantedEffect.ID()] = true
		}
	}
	if !seen["VaalColdSnap"] {
		t.Fatal("control: VaalColdSnap left the active skill list; the fixture is not measuring the secondary")
	}
	if !seen["ColdSnapAltX"] {
		t.Errorf("ColdSnapAltX is absent from the active skill list: the secondary never arrived. present=%v", seen)
	}
	if seen["ColdSnap"] {
		t.Error("plain ColdSnap is still in the active skill list: the BASE row's secondary was used, " +
			"which is the defect (it carries ChillingArea/NonHitChill that ColdSnapAltX does not)")
	}
}
