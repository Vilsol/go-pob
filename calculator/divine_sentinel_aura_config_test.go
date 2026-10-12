package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestDivineSentinelAuraConfigsSeedTheirConditions pins ConfigOptions.lua:382-406 -- the six "Divine Sentinel Auras" checkboxes.
func TestDivineSentinelAuraConfigsSeedTheirConditions(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)

	type wants struct {
		build string
		conds []string
	}
	cases := []wants{
		{"0011-saboteur-cyclone", []string{
			"Condition:DivineSentinelRegenLife",
			"Condition:DivineSentinelRegenMana",
			"Condition:DivineSentinelChaosResistance",
			"Condition:DivineSentinelSelfCurseEffect",
		}},
		{"0720-guardian-dominating-blow-of-inspiring-3", []string{
			"Condition:DivineSentinelPhysAsFire",
			"Condition:DivineSentinelPhysAsLightning",
			"Condition:DivineSentinelChaosResistance",
		}},
		{"1099-guardian-vaal-domination-2", []string{
			"Condition:DivineSentinelPhysAsFire",
			"Condition:DivineSentinelPhysAsLightning",
			"Condition:DivineSentinelRegenLife",
			"Condition:DivineSentinelRegenMana",
			"Condition:DivineSentinelChaosResistance",
			"Condition:DivineSentinelSelfCurseEffect",
		}},
		{"1152-guardian-dominating-blow-of-inspiring-5", []string{
			"Condition:DivineSentinelPhysAsFire",
			"Condition:DivineSentinelPhysAsLightning",
		}},
	}
	const controlBuild = "1064-guardian-dominating-blow-of-inspiring-4"
	allConds := []string{
		"Condition:DivineSentinelPhysAsFire",
		"Condition:DivineSentinelPhysAsLightning",
		"Condition:DivineSentinelRegenLife",
		"Condition:DivineSentinelRegenMana",
		"Condition:DivineSentinelChaosResistance",
		"Condition:DivineSentinelSelfCurseEffect",
	}

	load := func(name string) *Environment {
		raw, err := os.ReadFile(filepath.Join(corpusDir, name+".xml"))
		if err != nil {
			t.Fatalf("REFUSING: %s: %v", name, err)
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatalf("REFUSING: %s: %v", name, err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.ModDB == nil {
			t.Fatalf("REFUSING: %s produced no player modDB", name)
		}
		return env
	}

	set := 0
	for _, c := range cases {
		env := load(c.build)
		for _, cond := range c.conds {
			if !env.Player.ModDB.Flag(nil, cond) {
				t.Errorf("%s: player modDB lacks %s", c.build, cond)
				continue
			}
			set++
			if env.Minion != nil && env.Minion.ModDB != nil && !env.Minion.ModDB.Flag(nil, cond) {
				t.Errorf("%s: minion modDB lacks %s -- ConfigOptions.lua writes both halves", c.build, cond)
			}
		}
	}
	if set < 15 {
		t.Errorf("only %d of 15 flags reached the player; the fixture population has moved", set)
	}

	ctrl := load(controlBuild)
	for _, cond := range allConds {
		if ctrl.Player.ModDB.Flag(nil, cond) {
			t.Errorf("CONTROL %s sets no Divine Sentinel input and still carries %s -- the "+
				"handler is unconditional", controlBuild, cond)
		}
	}
}
