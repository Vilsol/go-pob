package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

var reTITestFunc = regexp.MustCompile(`(?m)^func ((?:Test|Fuzz)\w*)\(`)

// TestTestTiersSlowListNamesExistingTests: a renamed or deleted test left on the list would leave its successor in the fast tier unnoticed.
func TestTestTiersSlowListNamesExistingTests(t *testing.T) {
	repo, err := root.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := tiReadList(filepath.Join(repo, tiListFile))
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]map[string]bool{}
	for _, e := range entries {
		if _, ok := declared[e.pkg]; ok {
			continue
		}
		declared[e.pkg] = map[string]bool{}
		files, _ := filepath.Glob(filepath.Join(repo, e.pkg, "*_test.go"))
		for _, f := range files {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range reTITestFunc.FindAllSubmatch(src, -1) {
				declared[e.pkg][string(m[1])] = true
			}
		}
	}
	for _, e := range entries {
		if !declared[e.pkg][e.name] {
			t.Errorf("%s lists %s %s, which no _test.go there declares", tiListFile, e.pkg, e.name)
		}
		if e.seconds < tiSlowSeconds {
			t.Errorf("%s %s listed at %.1fs, under the %gs line", e.pkg, e.name, e.seconds, tiSlowSeconds)
		}
	}
}

func TestTestTiersScanKeepsTopLevelTimesAndPrintsOnlyFailures(t *testing.T) {
	log := strings.Join([]string{
		`{"Action":"build-output","ImportPath":"m/broken","Output":"broken.go:1: syntax\n"}`,
		`{"Action":"run","Package":"github.com/Vilsol/go-pob/a","Test":"TestA"}`,
		`{"Action":"output","Package":"github.com/Vilsol/go-pob/a","Test":"TestA","Output":"quiet\n"}`,
		`{"Action":"output","Package":"github.com/Vilsol/go-pob/a","Test":"TestA/sub","Output":"sub detail\n"}`,
		`{"Action":"pass","Package":"github.com/Vilsol/go-pob/a","Test":"TestA/sub","Elapsed":99}`,
		`{"Action":"pass","Package":"github.com/Vilsol/go-pob/a","Test":"TestA","Elapsed":12.5}`,
		`{"Action":"pass","Package":"github.com/Vilsol/go-pob/a","Test":"TestP/x","Elapsed":30}`,
		`{"Action":"pass","Package":"github.com/Vilsol/go-pob/a","Test":"TestP","Elapsed":0}`,
		`{"Action":"output","Package":"github.com/Vilsol/go-pob/a","Test":"TestB","Output":"boom\n"}`,
		`{"Action":"fail","Package":"github.com/Vilsol/go-pob/a","Test":"TestB","Elapsed":0.2}`,
		`{"Action":"output","Package":"github.com/Vilsol/go-pob/a","Output":"FAIL\n"}`,
		`{"Action":"fail","Package":"github.com/Vilsol/go-pob/a","Elapsed":13}`,
		`{"Action":"pass","Package":"github.com/Vilsol/go-pob/b","Elapsed":1}`,
	}, "\n")
	var out strings.Builder
	times, failed, err := tiScan(strings.NewReader(log), &out)
	if err != nil {
		t.Fatal(err)
	}
	if failed != 1 || len(times) != 3 || times[[2]string{"a", "TestA"}] != 99 || times[[2]string{"a", "TestP"}] != 30 {
		t.Fatalf("failed %d times %v", failed, times)
	}
	got := out.String()
	for _, want := range []string{"broken.go:1: syntax", "boom", "FAIL\ta\t13.0s", "ok  \tb\t1.0s"} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s", want, got)
		}
	}
	for _, not := range []string{"quiet", "sub detail"} {
		if strings.Contains(got, not) {
			t.Errorf("a passing test's output %q was printed:\n%s", not, got)
		}
	}
}

func TestTestTiersRecordKeepsOnlyTestsOverTheLine(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "run.jsonl")
	log := `{"Action":"pass","Package":"github.com/Vilsol/go-pob/a","Test":"TestSlow","Elapsed":10}
{"Action":"pass","Package":"github.com/Vilsol/go-pob/a","Test":"TestFast","Elapsed":9.9}
`
	if err := os.WriteFile(logPath, []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(dir, "slow.txt")
	if rc := tiRecord(list, []string{logPath}); rc != 0 {
		t.Fatalf("rc %d", rc)
	}
	entries, err := tiReadList(list)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].name != "TestSlow" || entries[0].pkg != "a" {
		t.Fatalf("entries %+v", entries)
	}
	if rc := tiRecord(list, []string{filepath.Join(dir, "empty")}); rc != root.RefusalCode {
		t.Fatalf("a missing log: rc %d", rc)
	}
}
