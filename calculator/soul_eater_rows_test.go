package calculator

import (
	"os"
	"testing"
)

// TestSoulEaterGrantRowsParse pins ModParser.lua:4304-4308.
func TestSoulEaterGrantRowsParse(t *testing.T) {
	for _, line := range []string{
		"When 90% of your Hex's Duration Expires on an Enemy, Eat 1 Soul per Enemy Power",
		"Eat 3 Souls when you Kill a Rare or Unique Enemy with this Weapon",
		"25% chance to gain Soul Eater for 5 seconds on Killing Blow against Rare and Unique Enemies with Double Strike or Dual Strike",
		"Eat a Soul when you Hit a Unique Enemy, no more than once every second",
	} {
		e := ParseMod(line, false)
		if e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "Condition:CanHaveSoulEater" || len(e.ModList[0].Tags()) != 0 {
			t.Errorf("%q: remainder %q, mods %v", line, e.Extra, e.ModList)
		}
	}
}

// TestSoulEaterStacksReachSpeed is the end-to-end on a Coiling Whisper carrier.
func TestSoulEaterStacksReachSpeed(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "0364-assassin-voltaxic-burst.xml")
	if got := env.Player.Output["Speed"]; got < 15.25*0.995 || got > 15.25*1.005 {
		t.Errorf("Speed = %v, PoB's oracle has 15.25", got)
	}
}
