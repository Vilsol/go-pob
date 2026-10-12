package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestComparePowerEquipsAbyssJewels: CompareTab.lua:2706-2758: an item row whose item has abyssal sockets is measured with the compared build's jewels in them.
func TestComparePowerEquipsAbyssJewels(t *testing.T) {
	initItemsTabData(t)
	a := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	b := loadCorpusBuild(t, "0028-occultist-forbidden-rite-of-soul-sacrifice.xml")
	before, err := builds.SerializeBuild(a)
	if err != nil {
		t.Fatal(err)
	}
	builder := NewComparePowerBuilder(a, b, []string{"Life", "FullDPS"}, ComparePowerCategories{Items: true}, nil)
	for builder.Step(1<<62) < 1 {
	}
	var withJewels []ComparePowerEntry
	for _, e := range builder.Report() {
		if strings.HasSuffix(e.Name, " Jewel") || strings.HasSuffix(e.Name, " Jewels") {
			withJewels = append(withJewels, e)
		}
	}
	if len(withJewels) == 0 {
		t.Fatalf("no item row measured with its abyssal jewels: %+v", builder.Report())
	}
	after, err := builds.SerializeBuild(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("the primary build changed: its abyssal sockets or items were not restored")
	}
	for _, e := range withJewels {
		cSet, cByID := setItems(b)
		item := ParseItem(cByID[setSlotItemID(cSet, e.SlotName)])
		item.NormaliseQuality()
		bare := builder.impactOf(builder.calc(&CalcOverride{RepSlotName: e.SlotName, RepItem: item}))
		if bare[0] == e.Impact[0] && bare[1] == e.Impact[1] {
			t.Errorf("%s: the jewels changed nothing (%v)", e.Name, e.Impact)
		}
	}
	t.Logf("%d item rows measured with abyssal jewels, e.g. %q", len(withJewels), withJewels[0].Name)
}
