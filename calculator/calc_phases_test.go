package calculator

import (
	"slices"
	"testing"
)

func TestBuildOutputNamesItsPhases(t *testing.T) {
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	c := &Calculator{PoB: build}
	var phases []string
	c.OnPhase(func(name string) { phases = append(phases, name) })
	c.BuildOutput(OutputModeMain)
	want := []string{"Setting up the items, tree and skills", "Calculating the main skill", "Calculating Full DPS"}
	if !slices.Equal(phases, want) {
		t.Fatalf("phases = %q, want %q", phases, want)
	}
}
