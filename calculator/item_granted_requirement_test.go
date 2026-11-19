package calculator

import (
	"strings"
	"testing"
)

// CalcSetup.lua:1500-1506 grant an influence-gated curse only while enough items of that influence are equipped.
func TestItemGrantedCurseNeedsInfluence(t *testing.T) {
	requireGameData(t, "3.30")
	for _, equipped := range []bool{false, true} {
		build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
		text := "Rarity: RARE\nTest Amulet\nPaua Amulet\n"
		if equipped {
			text += "Hunter Item\n"
		}
		text += "Implicits: 0\nCurse Enemies with Vulnerability on Hit if 1 Hunter Items are Equipped"
		build.SetSlotItem("Amulet", build.AddItem(text))
		BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		count := 0
		for _, group := range build.Skills.SkillSets[activeSkillSetIndex(build.Skills)].Skills {
			if strings.HasPrefix(group.Source, "Item:") && strings.HasSuffix(group.Source, ":Test Amulet, Paua Amulet") &&
				len(group.Gems) > 0 && group.Gems[0].SkillID == "Vulnerability" {
				count++
			}
		}
		want := 0
		if equipped {
			want = 1
		}
		if count != want {
			t.Errorf("Hunter item equipped %v: %d Vulnerability groups, want %d", equipped, count, want)
		}
	}
}
