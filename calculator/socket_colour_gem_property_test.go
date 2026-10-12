package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

// TestSocketColourGemPropertyReachesTheGemInThatSocket pins all four ends of PoB's per-gem GemProperty tabulation.
func TestSocketColourGemPropertyReachesTheGemInThatSocket(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build              string
		wantEffectiveLevel int
		wantLevelReq       int
		wantMinionLevel    int
		wantEmpowerLevel   int
	}{
		{"0611-necromancer-summon-holy-relic-of-conviction-8.xml", 26, 82, 82, 6},
		{"1115-necromancer-summon-holy-relic-of-conviction-16.xml", 31, 91, 91, 6},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			ms := env.Player.MainSkill
			if ms == nil || ms.ActiveEffect == nil || ms.ActiveEffect.SrcInstance == nil {
				t.Fatal("premise broken: no main skill with a source gem instance")
			}
			ae := ms.ActiveEffect

			if got := ae.Level; got != tc.wantEffectiveLevel {
				t.Errorf("effective gem level = %d, want %d (raw %d)", got, tc.wantEffectiveLevel,
					ae.SrcInstance.Level)
			}
			if got := ae.GrantedEffectLevel.LevelRequirement; got != tc.wantLevelReq {
				t.Errorf("levelRequirement = %d, want %d", got, tc.wantLevelReq)
			}
			if ms.Minion == nil {
				t.Fatal("premise broken: the main skill summons no minion")
			}
			if got := ms.Minion.Level; got != tc.wantMinionLevel {
				t.Errorf("minion level = %d, want %d — this is minionMeta.level from the "+
					"committed dump, and it indexes EVERY minion table", got, tc.wantMinionLevel)
			}

			empower := 0
			for _, se := range ms.EffectList {
				if se.GrantedEffect != nil && se.GrantedEffect.ID() == "SupportEmpower" {
					empower = se.Level
				}
			}
			if empower != tc.wantEmpowerLevel {
				t.Errorf("Empower's own effective level = %d, want %d — Dialla's +2 reaches the "+
					"SUPPORT, and the support pass tabulates GemProperty against its own gem cfg",
					empower, tc.wantEmpowerLevel)
			}
		})
	}
}

// TestSocketColourTagDropsAtGroupLevel is the negative half, and it is what says the fix did not simply apply the mod to every gem in the slot.
func TestSocketColourTagDropsAtGroupLevel(t *testing.T) {
	db := moddb.NewModDB()
	db.AddMod(mod.NewList("GemProperty", mod.GemProperty{
		Keyword: utils.Ptr("all"), Key: "level", Value: 2,
	}).Tag(mod.SocketedIn("Body Armour").SocketColor("R")))

	groupList := db.List(&moddb.ListCfg{SlotName: "Body Armour"}, "GemProperty")
	if len(groupList) != 0 {
		t.Errorf("a socket-colour GemProperty survived the GROUP-level cfg (%d mods); it would "+
			"then apply to every gem in the slot regardless of socket", len(groupList))
	}

	red := db.List(&moddb.ListCfg{SlotName: "Body Armour", SocketColor: "R"}, "GemProperty")
	if len(red) != 1 {
		t.Errorf("the mod did not reach a RED socket cfg (%d mods, want 1)", len(red))
	}
	green := db.List(&moddb.ListCfg{SlotName: "Body Armour", SocketColor: "G"}, "GemProperty")
	if len(green) != 0 {
		t.Errorf("the mod reached a GREEN socket cfg (%d mods, want 0)", len(green))
	}
	other := db.List(&moddb.ListCfg{SlotName: "Helmet", SocketColor: "R"}, "GemProperty")
	if len(other) != 0 {
		t.Errorf("the mod reached a red socket in another slot (%d mods, want 0)", len(other))
	}
}
