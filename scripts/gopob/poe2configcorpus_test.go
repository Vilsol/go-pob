package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPoe2ConfigCorpusValueNeverDrawsTheDefault(t *testing.T) {
	t.Parallel()
	list := p2ccOption{Var: "l", Type: "list", DefaultIndex: 2, List: []string{"s:A", "s:B", "n:-60"}}
	seen := map[string]bool{}
	r := p2ccRand(1, "b")
	for range 200 {
		attr, v := p2ccValue(list, r)
		if v == "B" {
			t.Fatalf("drew the default list entry")
		}
		seen[attr+"="+v] = true
	}
	if want := map[string]bool{"string=A": true, "number=-60": true}; !reflect.DeepEqual(seen, want) {
		t.Fatalf("list draws %v, want %v", seen, want)
	}
	if attr, v := p2ccValue(p2ccOption{Type: "check", DefaultState: "b:true"}, r); attr != "boolean" || v != "false" {
		t.Fatalf("a default-on check drew %s=%s", attr, v)
	}
	if _, v := p2ccValue(p2ccOption{Type: "check"}, r); v != "true" {
		t.Fatalf("a default-off check drew %s", v)
	}
}

func TestPoe2ConfigCorpusRewriteTargetsTheActiveSetAndKeepsExistingInputs(t *testing.T) {
	t.Parallel()
	xml := `<Config activeConfigSet="2">
		<ConfigSet id="1" title="a">
		</ConfigSet>
		<ConfigSet id="2" title="b">
			<Input name="enemyLevel" number="80"/>
		</ConfigSet>
	</Config>`
	out, kept, err := p2ccRewrite(xml, []p2ccSet{{"enemyLevel", "number", "1"}, {"bossName", "string", `Xesht's "Wrath" & co`}})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 || kept[0].Var != "bossName" {
		t.Fatalf("kept %v, want only bossName", kept)
	}
	set2 := out[strings.Index(out, `<ConfigSet id="2"`):]
	if !strings.Contains(set2, `<Input name="bossName" string="Xesht&apos;s &quot;Wrath&quot; &amp; co"/>`) {
		t.Fatalf("input not in the active set or not escaped:\n%s", out)
	}
	if strings.Count(out, "bossName") != 1 || strings.Count(out, `name="enemyLevel"`) != 1 {
		t.Fatalf("rewrite duplicated an input:\n%s", out)
	}
	if _, _, err := p2ccRewrite(`<Config activeConfigSet="1"><ConfigSet id="1"/></Config>`, nil); err == nil {
		t.Fatal("an empty ConfigSet was rewritten")
	}
}

func TestPoe2ConfigCorpusSampleIsSeededAndSorted(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	for _, n := range []string{"c1", "c2", "c3", "c4", "c5", "c6"} {
		if err := os.WriteFile(filepath.Join(d, n+".xml"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	a, err := p2ccSample(d, 3, 7)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := p2ccSample(d, 3, 7)
	c, _ := p2ccSample(d, 3, 8)
	if !reflect.DeepEqual(a, b) || len(a) != 3 {
		t.Fatalf("seed 7 gave %v then %v", a, b)
	}
	if reflect.DeepEqual(a, c) {
		t.Fatalf("seeds 7 and 8 drew the same sample %v", a)
	}
	for i := 1; i < len(a); i++ {
		if a[i-1] >= a[i] {
			t.Fatalf("sample %v is not sorted", a)
		}
	}
}

func TestPoe2ConfigCorpusParseOptionsDedupesAndFiltersQuests(t *testing.T) {
	t.Parallel()
	tsv := "index\tvar\ttype\tdefaultState\tdefaultIndex\tlist\tpredicates\tifSkill\tapplyLine\tlabel\n" +
		"2\tresistancePenalty\tlist\t\t7\tn:0\x1fn:-60\t\t\t0\tRes\n" +
		"3\tx\tcheck\t\t\t\tifCond=Y\t\t12\tX\n" +
		"4\tx\tcheck\t\t\t\t\t\t13\tX again\n" +
		"5\tquestAct 1Foo\tlist\t\t1\ts:None\x1fs:Bar\t\t\t20\tQ\n"
	opts, err := p2ccParseOptions(tsv)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 3 || opts[1].ApplyLine != 12 || len(opts[0].List) != 2 {
		t.Fatalf("parsed %+v", opts)
	}
	var elig []string
	for _, o := range opts {
		if p2ccEligible(o) {
			elig = append(elig, o.Var)
		}
	}
	if want := []string{"resistancePenalty", "x"}; !reflect.DeepEqual(elig, want) {
		t.Fatalf("eligible %v, want %v", elig, want)
	}
	if _, err := p2ccParseOptions("h\n1\tshort\n"); err == nil {
		t.Fatal("a short row parsed")
	}
}
