package exposition

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/storage"
)

// TestPrintModDB: ModDB.lua:323-358: mods by sorted name, then the true conditions and the positive multipliers, sorted.
func TestPrintModDB(t *testing.T) {
	db := moddb.NewModDB()
	db.AddMod(mod.NewFloat("Life", mod.TypeBase, 10).Source("Tree:123").Flag(mod.MFlagAttack | mod.MFlagHit).
		KeywordFlag(mod.KeywordFlagAura).Tag(mod.Condition("Foo")))
	db.AddMod(mod.NewFloat("Armour", mod.TypeIncrease, 2.5))
	db.AddMod(mod.NewFlag("Onslaught", true).Source("Config"))
	db.Conditions["Bar"] = true
	db.Conditions["Off"] = false
	db.Conditions["Alpha"] = true
	db.Multipliers["Zed"] = 3.7
	db.Multipliers["Zero"] = 0

	var out strings.Builder
	printModDB(&out, db)
	want := `=== Modifiers ===
'Armour':
	2.5 = INC|-|-|-|?
'Life':
	10 = BASE|Attack,Hit|Aura|type=Condition/VarList={Foo}|Tree:123
'Onslaught':
	true = FLAG|-|-|-|Config
=== Conditions ===
Alpha
Bar
=== Multipliers ===
Zed = 3
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

// TestFormatModFlags: ModTools.lua:114-127: every name whose bits are all set, masks included, sorted.
func TestFormatModFlags(t *testing.T) {
	if got := formatModFlags(mod.MFlagArea | mod.MFlagProjectile); got != "Area,Projectile,SourceMask" {
		t.Errorf("got %q", got)
	}
	if got := formatModFlags(mod.MFlagArea); got != "Area" {
		t.Errorf("got %q", got)
	}
	if got := formatModFlags(0); got != "-" {
		t.Errorf("got %q", got)
	}
}

// TestModDebugDump: Calcs.lua:37-70 on a real build: every section header, the main skill's effects, and the output table.
func TestModDebugDump(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, "0008-necromancer-vaal-summon-skeletons.xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	got := ModDebugDump(calculator.NewCalculator(*build))
	for _, want := range []string{
		"=== Modifiers ===", "=== Minion Mod DB ===", "=== Enemy Mod DB ===", "=== Main Skill ===",
		"=== Main Skill Flags ===\nMod: ", "\nKeyword: ", "=== Main Skill Mods ===", "=== Main Skill Data ===",
		"== Aux Skills ==", "== Output Table ==", "\nLife = ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("dump lacks %q", want)
		}
	}
	if ModDebugDump(nil) != "" || modDebugDump(nil) != "" {
		t.Error("a nil env must dump nothing")
	}
}

// TestItemTooltipModDebug: ItemsTab.lua:5160-5166: with Dev mode on.
func TestItemTooltipModDebug(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	build := corpusBuild(t, "0280-elementalist-fireball.xml")
	defer SetItemModDebug(false)

	off, err := ParseItem(build, pastedHelmet, calculator.ItemTooltipStats{})
	if err != nil {
		t.Fatal(err)
	}
	SetItemModDebug(true)
	on, err := ParseItem(build, pastedHelmet, calculator.ItemTooltipStats{})
	if err != nil {
		t.Fatal(err)
	}
	lines := on.Tooltip.Lines
	first := len(off.Tooltip.Lines)
	if first < len(lines) && lines[first].Separator {
		first++
	}
	extra := lines[first:]
	if len(extra) < 2 || !lines[first-1].Separator {
		t.Fatalf("dev lines %v", extra)
	}
	for i := 1; i < len(lines); i++ {
		if lines[i].Separator && lines[i-1].Separator {
			t.Fatalf("back-to-back separators at %d", i)
		}
	}
	if !slices.Contains(extra, calculator.TooltipLine{Text: "^780 = Life|BASE|-|-|-"}) {
		t.Errorf("no +80 Life line among %v", extra)
	}
}
