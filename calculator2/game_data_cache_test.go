package calculator2

import (
	"context"
	"reflect"
	"testing"

	game "github.com/Vilsol/go-pob-data/poe2"

	"github.com/Vilsol/go-pob/data/poe2minions"
	"github.com/Vilsol/go-pob/data/poe2skills"
)

// The .min assets zero only what the game data never reads, so it builds the same from them as from the full assets.
func TestMinTablesBuildAsTheFullTables(t *testing.T) {
	full, minCtx := context.Background(), gameDataContext()
	cache := gameDataCache()

	for _, table := range []string{game.GrantedEffectsTable, game.MonsterVarietiesTable, game.FlavourTextTable} {
		if _, ok := game.MinVariant(skillDataVersion, "raw/"+table); !ok {
			t.Fatalf("no .min of %s", table)
		}
	}

	build := func(ctx context.Context) (*poe2skills.Skills, map[string]map[string][]string, []game.CostType) {
		t.Helper()
		tables, err := game.LoadSkillTables(ctx, skillDataVersion, cache)
		if err != nil {
			t.Fatal(err)
		}
		flavourText, err := loadRows[game.FlavourText](ctx, game.FlavourTextTable, cache)
		if err != nil {
			t.Fatal(err)
		}
		melee, err := loadRows[game.CharacterMeleeSkill](ctx, game.CharacterMeleeSkillsTable, cache)
		if err != nil {
			t.Fatal(err)
		}
		costs, err := loadRows[game.CostType](ctx, game.CostTypesTable, cache)
		if err != nil {
			t.Fatal(err)
		}
		return poe2skills.Build(tables, flavourText), tables.CharacterMeleeSkills(melee), costs
	}
	wantSkills, wantMelee, wantCosts := build(full)
	gotSkills, gotMelee, gotCosts := build(minCtx)
	if !reflect.DeepEqual(gotSkills, wantSkills) {
		t.Error("the skills build differently from the .min tables")
	}
	if !reflect.DeepEqual(gotMelee, wantMelee) {
		t.Error("characterMeleeSkills builds differently from the .min tables")
	}
	for i := range wantCosts {
		if gotCosts[i].ID != wantCosts[i].ID || gotCosts[i].Divisor != wantCosts[i].Divisor {
			t.Errorf("cost type %d reads differently from the .min table", i)
		}
	}

	wantMinions, err := poe2minions.Load(full, skillDataVersion, cache)
	if err != nil {
		t.Fatal(err)
	}
	gotMinions, err := poe2minions.Load(minCtx, skillDataVersion, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(wantMinions) == 0 || !reflect.DeepEqual(gotMinions, wantMinions) {
		t.Errorf("the minions build differently from the .min tables (%d entries)", len(wantMinions))
	}
}
