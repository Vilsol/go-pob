package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// TestUtilityFlasksDisabledKeepsOnlyLifeAndManaFlasks pins CalcPerform.lua:1722, :1733-1735.
func TestUtilityFlasksDisabledKeepsOnlyLifeAndManaFlasks(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)
	file, err := os.ReadFile(filepath.Join(corpusDir, "1225-deadeye-scourge-arrow-of-menace-2.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	for si := range build.Items.ItemSets {
		for i := range build.Items.ItemSets[si].Slots {
			if strings.HasPrefix(build.Items.ItemSets[si].Slots[i].Name, "Flask") {
				build.Items.ItemSets[si].Slots[i].Active = true
			}
		}
	}
	title, enabled := "t", "true"
	build.Config.CustomModifierBlocks = append(build.Config.CustomModifierBlocks, pob.CustomModifierBlock{
		Title: &title, Enabled: &enabled, Text: "Life Flask Effects are not removed when Unreserved Life is Filled",
	})
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if !env.ModDB.Flag(nil, "UtilityFlasksDoNotApplyToPlayer") {
		t.Fatal("CONTROL FAILED: UtilityFlasksDoNotApplyToPlayer is not set")
	}

	utility := 0
	for _, item := range env.Flasks {
		if item.Base == nil || item.Base.Flask == nil || item.Base.Flask.Life > 0 || item.Base.Flask.Mana > 0 || len(item.Mods) == 0 {
			continue
		}
		utility++
		source := item.Mods[0].GetSource()
		for name, mods := range env.ModDB.Mods {
			for _, m := range mods {
				if m.GetSource() == source {
					t.Errorf("%s: %s from utility flask %q reached the player", name, m.Type(), source)
				}
			}
		}
	}
	if utility == 0 {
		t.Fatal("CONTROL FAILED: no active utility flask with mods")
	}

	haveLifeFlask := false
	for _, item := range env.Flasks {
		if item.Base != nil && item.Base.Flask != nil && item.Base.Flask.Life > 0 {
			haveLifeFlask = true
		}
	}
	if !haveLifeFlask {
		t.Fatal("CONTROL FAILED: no active life flask")
	}
	if !env.ModDB.Flag(nil, "LifeFlaskEffectNotRemoved") {
		t.Fatal("CONTROL FAILED: LifeFlaskEffectNotRemoved did not land")
	}
	if env.ModDB.Sum(mod.TypeBase, nil, "LifeRecovery") <= 0 {
		t.Error("no LifeRecovery from the life flask")
	}
}
