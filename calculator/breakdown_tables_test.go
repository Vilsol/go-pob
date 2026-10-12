package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

var (
	convDstRe = regexp.MustCompile(`^\d+% to (Physical|Lightning|Cold|Fire|Chaos)$`)
	gainDstRe = regexp.MustCompile(`^\d+% gained as (Physical|Lightning|Cold|Fire|Chaos)$`)
)

// TestBreakdownTablesCarryPoBsColumns: CalcOffence.lua:99-131: a damage type that converts or gains into another names how much.
func TestBreakdownTablesCarryPoBsColumns(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join("..", "testdata", "corpus-3.29", "00[0-9][0-9]-*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: no corpus builds (%v)", err)
	}
	conv, gain, reservations := 0, 0, 0
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			continue
		}
		env := (&Calculator{PoB: loadCorpusBuild(t, filepath.Base(f))}).BuildOutput(OutputModeMain)
		if env == nil || env.Player.Breakdown == nil {
			continue
		}
		for key, e := range env.Player.Breakdown.GetData() {
			for _, d := range e.DamageTypes {
				if d.ConvDst != nil {
					conv++
					if !convDstRe.MatchString(*d.ConvDst) {
						t.Errorf("%s %s: convDst %q", f, key, *d.ConvDst)
					}
				}
				if d.GainDst != nil {
					gain++
					if !gainDstRe.MatchString(*d.GainDst) {
						t.Errorf("%s %s: gainDst %q", f, key, *d.GainDst)
					}
				}
			}
			for _, r := range e.Reservations {
				reservations++
				if r.SkillName == "" || slices.Contains([]string{"VaalHaste", "VaalGrace", "VaalDiscipline"}, r.SkillName) {
					t.Errorf("%s %s: reservation names its skill %q, not its grantedEffect.name", f, key, r.SkillName)
				}
			}
		}
	}
	t.Logf("convDst %d, gainDst %d, reservation rows %d", conv, gain, reservations)
	if conv == 0 || reservations == 0 {
		t.Fatalf("REFUSING: the sample graded nothing (convDst %d, reservations %d)", conv, reservations)
	}
}
