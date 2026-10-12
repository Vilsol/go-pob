package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestAdornedCorruptedJewelEffect pins ModParser.lua (the JewelData row), CalcSetup.lua:752-758 and :1135-1141.
//
//	CalcSetup.lua:1135   an arm of the item-scaling elseif chain reads the multiplier and scales
func TestAdornedCorruptedJewelEffect(t *testing.T) {
	t.Run("the parser row produces the jewel data", func(t *testing.T) {
		requireGameData(t, "3.29")

		e := ParseMod("100% increased Effect of Jewel Socket Passive Skills containing Corrupted Magic Jewels", false)
		if e == nil {
			t.Fatal("REFUSING: the line does not parse at all")
		}
		found := false
		for _, m := range e.ModList {
			if m.Name() != "JewelData" {
				continue
			}
			v, ok := m.Value().List().(mod.JewelData)
			if ok && v.Key == "corruptedMagicJewelIncEffect" {
				found = true
			}
		}
		if !found {
			t.Errorf("no JewelData{corruptedMagicJewelIncEffect} mod; The Adorned's whole chain "+
				"starts here, and %d corpus builds carry the line", 226)
		}
	})

	t.Run("the multiplier reaches the build", func(t *testing.T) {
		if os.Getenv("CORPUS") != "true" {
			t.Skip("set CORPUS=true")
		}
		requireGameData(t, "3.29")
		dir, _ := corpusNamesForLoadedData(t)
		file, err := os.ReadFile(filepath.Join(dir, "0282-chieftain-incinerate-of-venting.xml"))
		if err != nil {
			t.Fatalf("%v", err)
		}
		if !strings.Contains(string(file), "containing Corrupted Magic Jewels") {
			t.Fatal("REFUSING: the fixture no longer carries The Adorned")
		}
		build, err := builds.ParseBuild(file)
		if err != nil {
			t.Fatalf("%v", err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if got := env.ModDB.Multipliers["CorruptedMagicJewelEffect"]; got == 0 {
			t.Errorf("CorruptedMagicJewelEffect is 0; CalcSetup.lua:753 publishes it as the slot " +
				"loop reaches The Adorned, and the corrupted-jewel arm at :1135 reads it")
		}
	})
}
