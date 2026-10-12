package calculator

import (
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"testing"
)

// TestWhatIfsLetTheGCMarkWorkerRun: js/wasm has no sysmon, so nothing preempts one long call.
func TestWhatIfsLetTheGCMarkWorkerRun(t *testing.T) {
	if runtime.GOARCH != "wasm" {
		t.Skip("a js/wasm property: natively the mark workers run on other threads, and under load the assists can finish a cycle first")
	}
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, helmet := activeSetSlot(build, "Helmet")
	var ids []string
	for _, g := range buildImplicitMods(ParseItem(helmet), "EATER") {
		for _, m := range g.Mods {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) < 20 {
		t.Fatalf("REFUSING: %d Eater implicits for the helmet", len(ids))
	}

	samples := []metrics.Sample{
		{Name: "/gc/cycles/total:gc-cycles"},
		{Name: "/cpu/classes/gc/mark/dedicated:cpu-seconds"},
		{Name: "/cpu/classes/gc/mark/idle:cpu-seconds"},
		{Name: "/cpu/classes/gc/mark/assist:cpu-seconds"},
	}
	read := func() (uint64, float64, float64) {
		metrics.Read(samples)
		for _, s := range samples {
			if s.Value.Kind() == metrics.KindBad {
				t.Fatalf("REFUSING: runtime metric %s is not supported", s.Name)
			}
		}
		return samples[0].Value.Uint64(), samples[1].Value.Float64() + samples[2].Value.Float64(), samples[3].Value.Float64()
	}

	defer debug.SetGCPercent(debug.SetGCPercent(10))
	cycles0, background0, assist0 := read()
	cycles, calcs := cycles0, 0
	for k := 0; calcs < 400 && cycles-cycles0 < 4; k = (k + 20) % (len(ids) - 19) {
		misc.ImplicitSortValues(helmet, "EATER", ids[k:k+20], "Life")
		calcs += 20
		cycles, _, _ = read()
	}
	cycles, background, assist := read()
	if cycles-cycles0 < 2 {
		t.Fatalf("REFUSING: only %d GC cycles in %d what-ifs, nothing to measure", cycles-cycles0, calcs)
	}
	if background-background0 <= 0 {
		t.Fatalf("%d GC cycles in %d what-ifs: background marking %.3f s, assists %.3f s — the mark worker never ran",
			cycles-cycles0, calcs, background-background0, assist-assist0)
	}
}
