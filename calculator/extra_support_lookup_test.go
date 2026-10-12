package calculator

import (
	"sort"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	raw "github.com/Vilsol/go-pob/data/raw"
)

// TestExtraSupportResolvesBothGrantedEffects is the invariant behind a MEASURED ZERO.
func TestExtraSupportResolvesBothGrantedEffects(t *testing.T) {
	requireGameData(t, "3.29")

	cases := map[string][]string{
		"cast on critical strike": {"SupportCastOnCriticalStrike", "SupportCastOnCritTriggered"},
		"cast while channelling":  {"SupportCastWhileChannelling", "SupportCastWhileChannellingTriggered"},
		"cast on melee kill":      {"SupportCastOnMeleeKill", "SupportCastOnMeleeKillTriggered"},
	}
	for name, want := range cases {
		got := extraSupportIDsForName(name)
		if len(got) != len(want) {
			t.Errorf("%q resolved to %v, want %v -- a support with a secondary granted effect must "+
				"yield BOTH; one id is the shape the \"Support\"+id rescue produces", name, got, want)
			continue
		}
		for i := range want {
			if got[i].skillID != want[i] {
				t.Errorf("%q resolved to %v, want %v (order is primary then secondary)", name, got, want)
				break
			}
		}
	}

	l := gemLookup()
	names := make([]string, 0, len(l.skillIDForName))
	for n := range l.skillIDForName {
		names = append(names, n)
	}
	sort.Strings(names)

	escapes := 0
	for _, name := range names {
		ids := extraSupportIDsForName(name)
		if len(ids) < 2 {
			continue
		}
		escapes++
	}
	if escapes == 0 {
		t.Fatal("REFUSING: not one support resolves to two granted effects, so this test no " +
			"longer exercises the case that distinguishes the gem lookup from env.go's " +
			"\"Support\"+id rescue. Either the lookup regressed or the data changed shape.")
	}
	t.Logf("supports resolving to a primary AND a secondary granted effect: %d", escapes)

	if got := extraSupportIDsForName("added fire damage"); len(got) != 1 || got[0].skillID != "SupportAddedFireDamage" {
		t.Errorf("control: %q resolved to %v, want exactly [SupportAddedFireDamage]", "added fire damage", got)
	}
	if poe.GrantedEffectByID("SupportCastOnCritTriggered") == nil {
		t.Error("control: SupportCastOnCritTriggered is not a granted effect at this pin; the " +
			"fixtures above are naming ids that do not exist")
	}
}

// TestExtraSupportSecondaryEffectIsGrantedAsASkill covers ModParser.lua:2090-2095.
func TestExtraSupportSecondaryEffectIsGrantedAsASkill(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		name        string
		wantSkillID string
		wantAsSkill bool
	}{
		{"impending doom", "DoomBlast", true},
		{"prismatic burst", "PrismaticBurst", true},
		{"kinetic instability", "TriggeredSupportKineticInstability", true},
		{"cast on critical strike", "SupportCastOnCritTriggered", false},
	} {
		got := extraSupportIDsForName(tc.name)
		if len(got) != 2 {
			t.Errorf("%q resolved to %d entries, want a primary and a secondary", tc.name, len(got))
			continue
		}
		if got[1].skillID != tc.wantSkillID {
			t.Errorf("%q secondary is %q, want %q", tc.name, got[1].skillID, tc.wantSkillID)
			continue
		}
		if got[1].asSkill != tc.wantAsSkill {
			t.Errorf("%q secondary %q: asSkill = %v, want %v", tc.name, tc.wantSkillID, got[1].asSkill, tc.wantAsSkill)
		}
		if got[0].asSkill {
			t.Errorf("%q primary %q was granted as a skill; :2089 grants the primary as a support "+
				"unconditionally", tc.name, got[0].skillID)
		}
	}
}

// TestHandSeededGemIDResolves is the end-to-end check on the one entry PoB writes into gemIdLookup by hand (ModParser.lua:2032-2034).
func TestHandSeededGemIDResolves(t *testing.T) {
	requireGameData(t, "3.29")

	const name = "power charge on critical strike"

	id := extraSkillIDForName(name)
	if id == "" {
		t.Fatalf("extraSkillIDForName(%q) resolved to nothing; the hand-seeded entry is gone", name)
	}
	if _, ok := raw.PoBSkills[id]; !ok {
		t.Fatalf("extraSkillIDForName(%q) = %q, which is not a key of raw.PoBSkills -- the seed "+
			"names a skill that does not exist and every consumer of it resolves to nothing", name, id)
	}

	got := extraSupportIDsForName(name)
	if len(got) == 0 {
		t.Fatalf("extraSupportIDsForName(%q) resolved to nothing", name)
	}
	if got[0].skillID != id {
		t.Errorf("extraSupportIDsForName(%q) primary = %q, want the seeded %q", name, got[0].skillID, id)
	}
}
