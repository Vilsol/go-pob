package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

var fullDPSBenchBuilds = []string{
	"1591-necromancer-summon-holy-relic-of-conviction-15.xml",
	"0001-necromancer-unknown.xml",
	"0008-necromancer-vaal-summon-skeletons.xml",
	"0280-elementalist-fireball.xml",
	"15355-necromancer-unknown.xml",
}

func initBenchData(tb testing.TB) {
	tb.Helper()
	if err := poe.InitializeAll(raw.DataContext(tb.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		tb.Fatal(err)
	}
}

func loadHeavyBuild(tb testing.TB, name string) *pob.PathOfBuilding {
	tb.Helper()
	dir, err := corpusDirForVersion(raw.LatestVersion)
	if err != nil {
		tb.Fatal(err)
	}
	xml, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		tb.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(xml)
	if err != nil {
		tb.Fatalf("REFUSING: %v", err)
	}
	return build
}

func BenchmarkBuildOutputHeavy(b *testing.B) {
	initBenchData(b)
	for _, name := range fullDPSBenchBuilds {
		b.Run(name[:strings.IndexByte(name, '-')], func(b *testing.B) {
			build := loadHeavyBuild(b, name)
			c := &Calculator{PoB: build}
			c.BuildOutput(OutputModeMain)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				c.BuildOutput(OutputModeMain)
			}
		})
	}
}

func BenchmarkCalcFullDPSHeavy(b *testing.B) {
	initBenchData(b)
	for _, name := range fullDPSBenchBuilds {
		b.Run(name[:strings.IndexByte(name, '-')], func(b *testing.B) {
			build := loadHeavyBuild(b, name)
			c := &Calculator{PoB: build}
			env, _, _, _ := InitEnvOverride(build, envCache, OutputModeCalculator, nil)
			PerformCalc(env)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				calcFullDPS(c, env, OutputModeCalculator, envCache, nil, nil)
			}
		})
	}
}

func BenchmarkMiscCalcNodeHeavy(b *testing.B) {
	initBenchData(b)
	for _, name := range fullDPSBenchBuilds {
		b.Run(name[:strings.IndexByte(name, '-')], func(b *testing.B) {
			build := loadHeavyBuild(b, name)
			m := GetMiscCalculator(build)
			override := &CalcOverride{RemoveNodes: build.Build.PassiveNodes[len(build.Build.PassiveNodes)-1:]}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				m.Calc(override)
			}
		})
	}
}

func BenchmarkMiscCalcFlaskHeavy(b *testing.B) {
	initBenchData(b)
	build := loadHeavyBuild(b, fullDPSBenchBuilds[0])
	_, text := activeSetSlot(build, "Flask 2")
	if text == "" {
		b.Fatal("REFUSING: no Flask 2")
	}
	m := GetMiscCalculator(build)
	override := &CalcOverride{ToggleFlask: ParseItem(text)}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		m.Calc(override)
	}
}
