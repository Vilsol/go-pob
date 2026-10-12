package raw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestDataFieldPopulationCensus answers one question for every field of every go-pob-data table the calculator reads: IS THIS FIELD EVER POPULATED ON ANY ROW?
func TestDataFieldPopulationCensus(t *testing.T) {
	if err := poe.InitializeAll(t.Context(), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := poe.InitializeAll(DataContext(context.Background()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
			t.Error(err)
		}
	})

	tables := []struct {
		name string
		rows any
	}{
		{"SkillGem", poe.SkillGems},
		{"GrantedEffect", poe.GrantedEffects},
		{"GrantedEffectStatSet", poe.GrantedEffectStatSets},
		{"GrantedEffectsPerLevel", poe.GrantedEffectsPerLevels},
		{"GrantedEffectStatSetsPerLevel", poe.GrantedEffectStatSetsPerLevels},
		{"GrantedEffectQualityStat", poe.GrantedEffectQualityStats},
		{"ActiveSkill", poe.ActiveSkills},
		{"BaseItemType", poe.BaseItemTypes},
		{"CostType", poe.CostTypes},
		{"Stat", poe.Stats},
		{"Mod", poe.Mods},
		{"MonsterVariety", poe.MonsterVarieties},
		{"DefaultMonsterStat", poe.DefaultMonsterStats},
		{"WeaponType", poe.WeaponTypes},
		{"ArmourType", poe.ArmourTypes},
		{"Flask", poe.Flasks},
		{"ItemClass", poe.ItemClasses},
	}

	type row struct {
		table, field string
		populated    int
		total        int
	}
	var rows []row //nolint:prealloc
	var never []string

	for _, table := range tables {
		slice := reflect.ValueOf(table.rows)
		if slice.Kind() != reflect.Slice {
			t.Fatalf("REFUSING: %s is a %s, not a slice; the sweep cannot walk it", table.name, slice.Kind())
		}
		if slice.Len() == 0 {
			t.Fatalf("REFUSING: table %s loaded 0 rows; every field would read as never-populated",
				table.name)
		}

		counts := map[string]int{}
		var order []string
		for i := 0; i < slice.Len(); i++ {
			elem := slice.Index(i)
			for elem.Kind() == reflect.Pointer {
				if elem.IsNil() {
					break
				}
				elem = elem.Elem()
			}
			if elem.Kind() != reflect.Struct {
				continue
			}
			walkFields(elem, "", func(name string, populated bool) {
				if _, seen := counts[name]; !seen {
					counts[name] = 0
					order = append(order, name)
				}
				if populated {
					counts[name]++
				}
			})
		}

		sort.Strings(order)
		for _, field := range order {
			rows = append(rows, row{table.name, field, counts[field], slice.Len()})
			if counts[field] == 0 {
				never = append(never, fmt.Sprintf("%s.%s (0 of %d)", table.name, field, slice.Len()))
			}
		}
	}

	var sawPopulated, sawNever bool
	for _, r := range rows {
		if r.populated == r.total && r.total > 0 {
			sawPopulated = true
		}
		if r.populated == 0 {
			sawNever = true
		}
	}
	if !sawPopulated {
		t.Fatal("REFUSING: not one field is populated on every row; the walker is broken, not the data")
	}
	if !sawNever {
		t.Log("note: no never-populated field found at all — if that is real it is good news, " +
			"but check the walker still descends into embedded structs")
	}
	for _, want := range []struct{ table, field string }{
		{"SkillGem", "BaseItemType"},
		{"GrantedEffect", "ID"},
	} {
		found := false
		for _, r := range rows {
			if r.table == want.table && r.field == want.field {
				found = true
				if r.populated == 0 {
					t.Errorf("CONTROL FAILED: %s.%s reports never-populated; it is known to be "+
						"populated, so the walker is misreading this shape", want.table, want.field)
				}
			}
		}
		if !found {
			t.Errorf("CONTROL FAILED: the sweep never visited %s.%s; the walker is not descending "+
				"into embedded structs", want.table, want.field)
		}
	}

	sort.Strings(never)
	t.Logf("DATA FIELD POPULATION CENSUS over %d tables, %d fields", len(tables), len(rows))
	t.Logf("NEVER POPULATED ON ANY ROW: %d fields", len(never))
	for _, n := range never {
		t.Logf("  %s", n)
	}

	if dir := os.Getenv("CENSUS_OUT"); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		var b strings.Builder
		b.WriteString("table\tfield\tpopulated\ttotal\n")
		for _, r := range rows {
			fmt.Fprintf(&b, "%s\t%s\t%d\t%d\n", r.table, r.field, r.populated, r.total)
		}
		p := filepath.Join(dir, "data_field_population.tsv")
		if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
			t.Fatalf("CENSUS_OUT: %v", err)
		}
		t.Logf("wrote %s", p)
	}
}

func walkFields(v reflect.Value, prefix string, visit func(name string, populated bool)) {
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		value := v.Field(i)

		if field.Anonymous && value.Kind() == reflect.Struct {
			walkFields(value, prefix, visit)
			continue
		}
		if !field.IsExported() {
			continue
		}

		name := prefix + field.Name
		switch value.Kind() {
		case reflect.Pointer, reflect.Interface:
			visit(name, !value.IsNil())
		case reflect.Slice, reflect.Map, reflect.String:
			visit(name, value.Len() > 0)
		case reflect.Bool:
			visit(name, value.Bool())
		case reflect.Struct:
			visit(name, !value.IsZero())
		default:
			visit(name, !value.IsZero())
		}
	}
}
