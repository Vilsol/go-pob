package builds

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var nameSpecNotGrantedEffect = map[string][2]string{
	"12729-ladder-Xaros980_0711.ПоПыткаПыткаВан.xml": {"Vaal Animate Weapon", "Vaal Animate Weapon of Ranged Arms"},
	"9175-ladder-BlueBird22_3165.Голубочкин.xml":     {"Vaal Smite", "Vaal Smite of Divine Judgement"},
}

// TestSavedBuildSummaryMatchesPoB: The build list's summary of a saved build against what the pinned PoB holds once it has loaded the same file.
func TestSavedBuildSummaryMatchesPoB(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "saved_build_summary_pob.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		col := strings.Split(line, "\t")
		if len(col) != 5 {
			t.Fatalf("bad row %q", line)
		}
		rows++
		raw, err := os.ReadFile(filepath.Join("..", "testdata", "corpus-3.29", col[0]))
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		got, err := SavedBuildSummary(string(raw))
		if err != nil {
			t.Errorf("%s: %v", col[0], err)
			continue
		}
		asc := col[2]
		if asc == "None" {
			asc = ""
		}
		level, _ := strconv.Atoi(col[3])
		want := SavedBuild{ClassName: col[1], Level: level, Ascendancy: asc, MainSkill: col[4]}
		if alt, ok := nameSpecNotGrantedEffect[col[0]]; ok {
			if want.MainSkill == alt[0] {
				want.MainSkill = alt[1]
			} else {
				t.Errorf("%s: PoB's label is %q now, not %q: drop the exception", col[0], want.MainSkill, alt[0])
			}
		}
		if got != want {
			t.Errorf("%s:\n got %+v\nwant %+v", col[0], got, want)
		}
	}
	if rows < 50 {
		t.Fatalf("REFUSING: %d rows graded", rows)
	}
}

// TestSavedBuildSummaryAgreesWithTheParse: The list read class and level off a full parse.
func TestSavedBuildSummaryAgreesWithTheParse(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "testdata", "corpus-3.29", "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: no corpus (%v)", err)
	}
	graded := 0
	for i := 0; i < len(files); i += 20 {
		raw, err := os.ReadFile(files[i])
		if err != nil {
			t.Fatal(err)
		}
		full, perr := ParseBuild(raw)
		got, serr := SavedBuildSummary(string(raw))
		if (perr == nil) != (serr == nil) {
			t.Errorf("%s: parse error %v, summary error %v", files[i], perr, serr)
			continue
		}
		if perr != nil {
			continue
		}
		graded++
		if got.ClassName != string(full.Build.ClassName) || got.Level != full.Build.Level {
			t.Errorf("%s: summary %s %d, parse %s %d", files[i], got.ClassName, got.Level, full.Build.ClassName, full.Build.Level)
		}
	}
	if graded < 500 {
		t.Fatalf("REFUSING: %d builds graded", graded)
	}
}

// TestSavedBuildSummaryRefusesWhatTheParseRefuses: A build with no <Spec> fails the parse (activeSpec), so it fails the summary too.
func TestSavedBuildSummaryRefusesWhatTheParseRefuses(t *testing.T) {
	const in = `<PathOfBuilding><Build level="90" className="Witch" ascendClassName="Occultist"/><Tree activeSpec="1"/></PathOfBuilding>`
	if _, err := ParseBuildStr(in); err == nil {
		t.Fatal("control: the parse took a build with no spec")
	}
	if _, err := SavedBuildSummary(in); err == nil {
		t.Error("the summary took a build the parse refuses")
	}
}
