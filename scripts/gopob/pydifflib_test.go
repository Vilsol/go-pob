package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

type ddCase struct {
	Name string   `json:"name"`
	A    []string `json:"a"`
	B    []string `json:"b"`
}

type ddRand struct{ s uint64 }

func (r *ddRand) next(n int) int {
	if n <= 0 {
		return 0
	}
	r.s = r.s*6364136223846793005 + 1442695040888963407
	return int((r.s >> 33) % uint64(n)) //nolint:gosec // n is a small positive literal bound
}

func ddCases() []ddCase {
	cases := []ddCase{
		{"both-empty", []string{}, []string{}},
		{"a-empty", []string{}, []string{"x", "y"}},
		{"b-empty", []string{"x", "y"}, []string{}},
		{"identical", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"disjoint", []string{"a", "b"}, []string{"y", "z"}},
		{"one-element-each", []string{"a"}, []string{"a"}},
		{"tie-for-longest", []string{"q", "x", "y", "q", "x", "y"}, []string{"x", "y"}},
		{"prefix-suffix", []string{"a", "b", "c", "d", "e"}, []string{"a", "z", "e"}},
		{"all-same-a", []string{"s", "s", "s", "s"}, []string{"s", "s"}},
	}
	r := &ddRand{s: 0x5eed}
	alpha := []string{"a", "b", "c", "d", "e", "f"}
	for c := range 40 {
		la, lb := r.next(14), r.next(14)
		a := make([]string, la)
		b := make([]string, lb)
		for i := range a {
			a[i] = alpha[r.next(len(alpha))]
		}
		for i := range b {
			b[i] = alpha[r.next(len(alpha))]
		}
		cases = append(cases, ddCase{fmt.Sprintf("gen%02d", c), a, b})
	}
	for _, n := range []int{199, 200, 205, 400} {
		a := make([]string, 0, n)
		b := make([]string, 0, n)
		for i := range n {
			if i%3 == 0 {
				a = append(a, "POP")
				b = append(b, "POP")
			} else {
				a = append(a, fmt.Sprintf("a%d", i))
				b = append(b, fmt.Sprintf("b%d", i%7))
			}
		}
		cases = append(cases, ddCase{fmt.Sprintf("autojunk%d", n), a, b})
	}
	{
		const n = 205
		a := make([]string, 0, n)
		b := make([]string, 0, n)
		for i := range n {
			switch {
			case i%2 == 0:
				a = append(a, "POP")
				b = append(b, "POP")
			case i == 7 || i == 51 || i == 99:
				a = append(a, "EXACTLY3")
				b = append(b, "EXACTLY3")
			case i == 61:
				a = append(a, "RARE")
				b = append(b, "RARE")
			default:
				a = append(a, fmt.Sprintf("a%d", i))
				b = append(b, fmt.Sprintf("b%d", i))
			}
		}
		cases = append(cases, ddCase{"autojunk-on-the-cut", a, b})
	}
	return cases
}

func ddAsk(t *testing.T, script string, cases []ddCase) []string {
	t.Helper()
	payload, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "python3", "-c", script)
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3: %v", err)
	}
	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
}

func TestPyDifflibRatioMatchesCPython(t *testing.T) {
	cases := ddCases()
	for _, autojunk := range []bool{true, false} {
		aj := "True"
		if !autojunk {
			aj = "False"
		}
		want := ddAsk(t, "import json,sys,difflib\n"+
			"for c in json.load(sys.stdin):\n"+
			"    m = difflib.SequenceMatcher(None, c['a'], c['b'], autojunk="+aj+")\n"+
			"    print('%r %r' % (m.ratio(), m.quick_ratio()))", cases)
		if len(want) != len(cases) {
			t.Fatalf("python printed %d lines for %d cases", len(want), len(cases))
		}
		for i, c := range cases {
			m := newPySeqMatcher(c.A, c.B, autojunk)
			got := fmt.Sprintf("%s %s", pyFloat(m.ratio()), pyFloat(m.quickRatio()))
			if got != want[i] {
				t.Errorf("autojunk=%v %s: ratio/quick_ratio\n  go     = %s\n  python = %s",
					autojunk, c.Name, got, want[i])
			}
		}
	}
}

func TestPyDifflibOpcodesMatchCPython(t *testing.T) {
	cases := ddCases()
	for _, autojunk := range []bool{true, false} {
		aj := "True"
		if !autojunk {
			aj = "False"
		}
		want := ddAsk(t, "import json,sys,difflib\n"+
			"for c in json.load(sys.stdin):\n"+
			"    m = difflib.SequenceMatcher(None, c['a'], c['b'], autojunk="+aj+")\n"+
			"    print('|'.join('(%r, %d, %d, %d, %d)' % o for o in m.get_opcodes()))", cases)
		for i, c := range cases {
			m := newPySeqMatcher(c.A, c.B, autojunk)
			parts := []string{}
			for _, o := range m.getOpcodes() {
				parts = append(parts, pyOpcodeString(o))
			}
			got := strings.Join(parts, "|")
			if got != want[i] {
				t.Errorf("autojunk=%v %s: opcodes\n  go     = %s\n  python = %s",
					autojunk, c.Name, got, want[i])
			}
		}
	}
}

// TestPyDifflibUnifiedDiffMatchesCPython: n IS SWEPT, because get_grouped_opcodes trims the first and last EQUAL run to n and splits any interior run longer than 2n.
func TestPyDifflibUnifiedDiffMatchesCPython(t *testing.T) {
	cases := ddCases()
	for _, n := range []int{0, 1, 3, 6} {
		want := ddAsk(t, fmt.Sprintf(
			"import json,sys,difflib\n"+
				"for c in json.load(sys.stdin):\n"+
				"    d = list(difflib.unified_diff(c['a'], c['b'], 'pin', 'candidate',"+
				" lineterm='', n=%d))\n"+
				"    print(json.dumps(d))", n), cases)
		for i, c := range cases {
			got := pyUnifiedDiff(c.A, c.B, "pin", "candidate", "", "", n, "")
			var wantList []string
			if err := json.Unmarshal([]byte(want[i]), &wantList); err != nil {
				t.Fatalf("%s: %v", c.Name, err)
			}
			if len(got) != len(wantList) {
				t.Errorf("n=%d %s: unified_diff length go=%d python=%d\n  go=%q\n  py=%q",
					n, c.Name, len(got), len(wantList), got, wantList)
				continue
			}
			for k := range got {
				if got[k] != wantList[k] {
					t.Errorf("n=%d %s line %d:\n  go     = %q\n  python = %q",
						n, c.Name, k, got[k], wantList[k])
					break
				}
			}
		}
	}
}
