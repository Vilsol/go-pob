package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

func TestNewBuildResolvesClassAndAscendancy(t *testing.T) {
	s := testServer(t)
	res := call(t, s, "new_build", map[string]any{"class": "witch", "ascendancy": "elementalist", "level": 70})
	sum := res["summary"].(map[string]any)
	if sum["class"] != "Witch" || sum["ascendancy"] != "Elementalist" || sum["level"] != float64(70) {
		t.Fatalf("summary = %v", sum)
	}
	if used := sum["points"].(map[string]any)["used"]; used != float64(0) {
		t.Fatalf("an empty build uses %v points", used)
	}
	if sum["main_skill"] != nil {
		t.Fatalf("an empty build has main skill %v", sum["main_skill"])
	}

	e := callErr(t, s, "new_build", map[string]any{"class": "Wizard"})
	if e.Code != CodeInvalidValue || len(e.Allowed) != 7 {
		t.Fatalf("unknown class: %s %v", e.Code, e.Allowed)
	}
	e = callErr(t, s, "new_build", map[string]any{"class": "Witch", "ascendancy": "Juggernaut"})
	if e.Code != CodeInvalidValue {
		t.Fatalf("foreign ascendancy: %s", e.Code)
	}
	e = callErr(t, s, "new_build", map[string]any{"class": "Witch", "colour": "red"})
	if e.Code != CodeInvalidArgument || len(e.Valid) == 0 {
		t.Fatalf("unknown property: %s %v", e.Code, e.Valid)
	}
}

func TestNewBuildForksAHandle(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	base := call(t, s, "get_stats", map[string]any{"handle": h})
	fork := call(t, s, "new_build", map[string]any{"from_handle": h})
	if fork["handle"] == h {
		t.Fatal("the fork reused the handle")
	}
	sum := fork["summary"].(map[string]any)
	if sum["class"] != "Marauder" || sum["level"] != float64(90) {
		t.Fatalf("fork summary = %v", sum)
	}
	if got := stat(sum, "Life"); got != stat(base, "Life") {
		t.Fatalf("fork Life %v, source %v", got, stat(base, "Life"))
	}
}

func TestImportBuildAcceptsFileXMLAndCode(t *testing.T) {
	s := testServer(t)
	raw, err := os.ReadFile(fixture("3.xml"))
	if err != nil {
		t.Fatal(err)
	}
	code, err := pob.CompressEncode(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	dps := make([]float64, 0, 3)
	for _, src := range []struct{ source, kind string }{{fixture("3.xml"), "file"}, {string(raw), "xml"}, {code, "code"}} {
		res := call(t, s, "import_build", map[string]any{"source": src.source})
		if res["source_kind"] != src.kind {
			t.Fatalf("source_kind %v, want %s", res["source_kind"], src.kind)
		}
		v := res["versions"].(map[string]any)
		if v["converted"] != true || v["build_tree"] != "3_18" {
			t.Fatalf("versions = %v", v)
		}
		dps = append(dps, stat(res["summary"].(map[string]any), "MainDPS"))
	}
	if dps[0] == 0 || dps[0] != dps[1] || dps[1] != dps[2] {
		t.Fatalf("MainDPS by source = %v", dps)
	}

	kept := call(t, s, "import_build", map[string]any{"source": fixture("3.xml"), "convert_tree": false})
	if v := kept["versions"].(map[string]any); v["converted"] != false || v["tree"] != "3_18" {
		t.Fatalf("convert_tree:false versions = %v", v)
	}

	if e := callErr(t, s, "import_build", map[string]any{"source": "not a build"}); e.Code != CodeParseError {
		t.Fatalf("junk source: %s", e.Code)
	}
	if e := callErr(t, s, "import_build", map[string]any{"source": `<PathOfBuilding2><Build level="1"/></PathOfBuilding2>`}); e.Code != CodeUnsupportedGame {
		t.Fatalf("PoE2 source: %s", e.Code)
	}
	if e := callErr(t, s, "import_build", map[string]any{"source": "https://example.com/build/1"}); e.Code != CodeImportFailed || len(e.Valid) == 0 {
		t.Fatalf("unknown site: %s %v", e.Code, e.Valid)
	}
}

func TestExportBuildWritesOneFilePerExport(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	first := call(t, s, "export_build", map[string]any{"handle": h})
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": []any{map[string]any{"op": "set_level", "level": 80}}})
	second := call(t, s, "export_build", map[string]any{"handle": h, "format": "xml"})
	if second["revision"] != float64(1) || second["xml"] == nil || second["code"] != nil {
		t.Fatalf("xml export = revision %v, keys %v", second["revision"], keys(second))
	}

	files, err := filepath.Glob(filepath.Join(s.opts.ExportDir, "*.xml"))
	if err != nil || len(files) != 2 {
		t.Fatalf("export files = %v (%v)", files, err)
	}
	if filepath.Base(files[0]) != "0001-"+h+"-r0.xml" || filepath.Base(files[1]) != "0002-"+h+"-r1.xml" {
		t.Fatalf("export file names = %v", files)
	}
	written, err := os.ReadFile(files[1])
	if err != nil {
		t.Fatal(err)
	}
	b, err := builds.ParseBuild(written)
	if err != nil || b.Build.Level != 80 || len(b.Build.PlayerStats) == 0 {
		t.Fatalf("the export does not read back: level %d, %d stats, %v", b.Build.Level, len(b.Build.PlayerStats), err)
	}

	again := call(t, s, "import_build", map[string]any{"source": first["code"]})
	if got, want := stat(again["summary"].(map[string]any), "MainDPS"), stat(first["summary"].(map[string]any), "MainDPS"); got != want {
		t.Fatalf("re-imported code MainDPS %v, exported %v", got, want)
	}
}

func TestRevertBuildMovesBetweenRevisions(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	totals := map[string]any{"handle": h, "stats": []string{"MainDPS", "FireResistTotal"}}
	base := call(t, s, "get_stats", totals)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": []any{map[string]any{"op": "set_bandit", "bandit": "Alira"}}})
	edited := call(t, s, "get_stats", totals)
	if stat(edited, "FireResistTotal") != stat(base, "FireResistTotal")+15 {
		t.Fatalf("helping Alira moved FireResistTotal %v -> %v", stat(base, "FireResistTotal"), stat(edited, "FireResistTotal"))
	}

	back := call(t, s, "revert_build", map[string]any{"handle": h})
	if back["revision"] != float64(0) || back["latest_revision"] != float64(1) {
		t.Fatalf("revert = %v", back)
	}
	if got := call(t, s, "get_stats", totals); stat(got, "FireResistTotal") != stat(base, "FireResistTotal") {
		t.Fatalf("reverted FireResistTotal %v, base %v", stat(got, "FireResistTotal"), stat(base, "FireResistTotal"))
	}
	call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 1})
	if got := call(t, s, "get_stats", totals); stat(got, "FireResistTotal") != stat(edited, "FireResistTotal") {
		t.Fatal("revision 1 was not kept")
	}
	if e := callErr(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 9}); e.Code != CodeInvalidValue {
		t.Fatalf("missing revision: %s", e.Code)
	}
	assertSize(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 0}, capWrite)
}

func TestUnknownHandleListsLiveHandles(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	e := callErr(t, s, "get_stats", map[string]any{"handle": "b99"})
	if e.Code != CodeUnknownHandle || len(e.ValidHandles) != 1 || e.ValidHandles[0] != h {
		t.Fatalf("unknown handle: %s %v", e.Code, e.ValidHandles)
	}
}

func TestToolsWaitForGameData(t *testing.T) {
	s := New(Options{})
	e := callErr(t, s, "get_build", map[string]any{})
	if e.Code != CodeDataLoading || e.RetryAfterMs == 0 {
		t.Fatalf("before data: %s", e.Code)
	}
}

func TestHandleLimitEvictsTheOldestUnexported(t *testing.T) {
	s := testServer(t)
	first := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	second := call(t, s, "new_build", map[string]any{"class": "Witch"})["handle"].(string)
	call(t, s, "export_build", map[string]any{"handle": first})
	for range maxHandles - 2 {
		call(t, s, "new_build", map[string]any{"class": "Witch"})
	}
	res := call(t, s, "new_build", map[string]any{"class": "Witch"})
	if w, _ := res["server_warnings"].([]any); len(w) != 1 {
		t.Fatalf("server_warnings = %v", res["server_warnings"])
	}
	if _, ok := s.handles[second]; ok {
		t.Fatal("the oldest unexported handle was kept")
	}
	if _, ok := s.handles[first]; !ok {
		t.Fatal("the exported handle was evicted")
	}
}
