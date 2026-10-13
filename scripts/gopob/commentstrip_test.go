package main

import (
	"strings"
	"testing"
)

func TestCommentStripKeepsWhatItMustAndDropsProse(t *testing.T) {
	pob := map[string]bool{csNorm("Merge keystone modifiers"): true}
	src := `package x

// Deliberately long prose that exists only in go-pob.
//
//pob:src/Modules/CalcPerform.lua:1-2
func A() int {
	// Merge keystone modifiers
	x := 1 // trailing prose
	//nolint:gosec // reason
	//	local y = modDB:Sum("BASE", nil, "Life")
	//	y := 2
	/*
		TODO Breakdown
		breakdown.AddLine("x")
	*/
	/*
	   ⚠️ An essay in a block comment.
	   It explains history.
	*/
	return x
}
`
	st := &csStats{lines: map[csClass]int{}, showAll: true}
	out, err := csStrip("x.go", []byte(src), pob, map[csClass]bool{csProse: true, csGoCode: true}, st, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, keep := range []string{"//pob:src/Modules/CalcPerform.lua:1-2", "// Merge keystone modifiers", "//nolint:gosec // reason",
		`//	local y = modDB:Sum("BASE", nil, "Life")`, "TODO Breakdown", "x := 1\n"} {
		if !strings.Contains(got, keep) {
			t.Errorf("dropped %q:\n%s", keep, got)
		}
	}
	for _, drop := range []string{"Deliberately long prose", "trailing prose", "y := 2", "An essay", "\n//\n"} {
		if strings.Contains(got, drop) {
			t.Errorf("kept %q:\n%s", drop, got)
		}
	}
}

func TestCommentStripMaxProseFlagsLongRunsOnly(t *testing.T) {
	src := `package x

// one
// two
// three
func A() {}

// one
// two
// three
// four
//pob:src/Modules/CalcPerform.lua:1-2
func B() {}
`
	st := &csStats{lines: map[csClass]int{}, maxProse: 3}
	if _, err := csStrip("x.go", []byte(src), map[string]bool{}, map[csClass]bool{}, st, 0); err != nil {
		t.Fatal(err)
	}
	if len(st.long) != 1 || st.long[0] != "x.go:8" {
		t.Errorf("long runs = %v, want exactly [x.go:8] (three lines is allowed, four is not)", st.long)
	}
}

func TestCommentStripTestsKeepsOneDocLinePerTest(t *testing.T) {
	src := `package x

// helper prose goes.
func helper() int { return 1 }

// TestA guards the first thing.
// Second line of history goes.
func TestA(t *testing.T) {
	// in-body prose goes.
	//go-only: kept marker
	_ = helper() // trailing prose goes
}

// pins the rounding position.
// More detail goes.
//
//pob:src/Modules/CalcPerform.lua:1-2
func BenchmarkB(b *testing.B) {}

/*
   FuzzC walks every input.
   It explains history.
*/
func FuzzC(f *testing.F) {}

// TestD joins a sentence that cites CalcDefence.lua:411
// across two lines. Its second sentence goes.
func TestD(t *testing.T) {}

// TestE pins a sentence that runs long, with a clause after the comma that carries on and on
// and on, well past the budget of one doc line, so it is cut at its first clause boundary.
func TestE(t *testing.T) {}

// TestF.
func TestF(t *testing.T) {}

// Item.lua:2262-2265 scales a weapon's local added elemental ranges by two increased mods that were
// never applied here.
func TestWeaponLocalElementalDamageScalesAddedRanges(t *testing.T) {}

// Testify is not a test name, so this goes.
func Testify() {}
`
	drop := map[csClass]bool{csProse: true, csGoCode: true}
	st := &csStats{lines: map[csClass]int{}, showAll: true, tests: true}
	out, err := csStrip("x_test.go", []byte(src), map[string]bool{}, drop, st, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, keep := range []string{"// TestA guards the first thing.\nfunc TestA", "\t//go-only: kept marker\n",
		"// BenchmarkB: pins the rounding position.\n//\n//pob:src/Modules/CalcPerform.lua:1-2\nfunc BenchmarkB",
		"// FuzzC walks every input.\nfunc FuzzC",
		"// TestD joins a sentence that cites CalcDefence.lua:411 across two lines.\nfunc TestD",
		"// TestE pins a sentence that runs long.\nfunc TestE", "// TestF.\nfunc TestF"} {
		if !strings.Contains(got, keep) {
			t.Errorf("missing %q:\n%s", keep, got)
		}
	}
	for _, gone := range []string{"helper prose", "Second line", "in-body", "trailing prose", "More detail", "history", "second sentence", "and on", "Testify is"} {
		if strings.Contains(got, gone) {
			t.Errorf("kept %q:\n%s", gone, got)
		}
	}
	if st.lines[csTestDoc] != 10 {
		t.Errorf("test-doc lines = %d, want 10 (the FuzzC block is four)", st.lines[csTestDoc])
	}
	if !strings.Contains(got, "// TestWeaponLocalElementalDamageScalesAddedRanges: Item.lua:2262-2265 scales") {
		t.Errorf("the prefixed line was cut at its own prefix:\n%s", got)
	}
	again, err := csStrip("x_test.go", out, map[string]bool{}, drop, &csStats{lines: map[csClass]int{}, tests: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != got {
		t.Errorf("a second pass changed the output:\n%s", again)
	}

	st = &csStats{lines: map[csClass]int{}, showAll: true}
	out, err = csStrip("x_test.go", []byte(src), map[string]bool{}, drop, st, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "guards the first thing") || st.lines[csTestDoc] != 0 {
		t.Errorf("without tests mode the doc line survived:\n%s", out)
	}
}
