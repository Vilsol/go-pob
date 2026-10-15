package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

// TestBalefulDominionRecoupRows pins ModParser.lua:439 and :4770 (upstream #10231).
func TestBalefulDominionRecoupRows(t *testing.T) {
	for line, want := range map[string]string{
		"Life Recoup also recovers Mana":     "AddLifeRecoupToManaRecoup/FLAG",
		"50% increased Recovery from Recoup": "RecoupRecoveryAmount/INC",
		"20% less Recovery from Recoup":      "RecoupRecoveryAmount/MORE",
	} {
		entry := ParseMod(line, false)
		if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 1 {
			t.Errorf("%q: extra %q, %d mods", line, entry.Extra, len(entry.ModList))
			continue
		}
		if got := entry.ModList[0].Name() + "/" + string(entry.ModList[0].Type()); got != want {
			t.Errorf("%q: %s, want %s", line, got, want)
		}
	}
}

// TestBalefulDominionRecoup pins CalcDefence.lua:1407-1429 (upstream #10231): Life Recoup also recovers Mana, and Recovery from Recoup scales every recoup.
func TestBalefulDominionRecoup(t *testing.T) {
	requireGameData(t, "3.29")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	file, err := os.ReadFile(filepath.Join(corpusDir, "0006-elementalist-storm-burst.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatal(err)
	}
	title, enabled := "t", "true"
	build.Config.CustomModifierBlocks = append(build.Config.CustomModifierBlocks, pob.CustomModifierBlock{
		Title: &title, Enabled: &enabled, Text: "Life Recoup also recovers Mana\n50% increased Recovery from Recoup",
	})
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	if got := env.Player.Output["LifeRecoup"]; got != 14*1.5 {
		t.Errorf("LifeRecoup = %v, want 14 * 1.5", got)
	}
	if got := env.Player.Output["ManaRecoup"]; got != 14*1.5*env.Player.Output["ManaRecoveryRateMod"] {
		t.Errorf("ManaRecoup = %v, want 14 * 1.5 * %v", got, env.Player.Output["ManaRecoveryRateMod"])
	}
}
