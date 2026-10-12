package gateversions

import (
	"os/exec"
	"strings"
	"testing"
)

func TestExactlyOneAuthoritativeRow(t *testing.T) {
	r, err := Authoritative()
	if err != nil {
		t.Fatal(err)
	}
	if r.Version == "" || r.OracleTest == "" || r.CorpusTest == "" {
		t.Fatalf("the authoritative row is incomplete: %+v", r)
	}
}

// TestAuthoritativeRefusesOnAnythingButExactlyOne: THE REFUSALS ARE THE POINT OF THIS TABLE.
func TestAuthoritativeRefusesOnAnythingButExactlyOne(t *testing.T) {
	orig := Table
	t.Cleanup(func() { Table = orig })

	for name, table := range map[string][]Row{
		"none": {{Version: "9.9", OracleTest: "A", CorpusTest: "B"}},
		"two": {
			{Version: "9.9", OracleTest: "A", CorpusTest: "B", Authoritative: true},
			{Version: "9.8", OracleTest: "C", CorpusTest: "D", Authoritative: true},
		},
		"empty": {},
	} {
		Table = table
		if _, err := Authoritative(); err == nil {
			t.Fatalf("%s: returned a row instead of refusing", name)
		}
	}
}

// TestAllTestsRefusesOnAnEmptyTable: AN EMPTY TABLE MUST NOT PRINT "".
func TestAllTestsRefusesOnAnEmptyTable(t *testing.T) {
	orig := Table
	t.Cleanup(func() { Table = orig })
	Table = nil
	if got, err := AllTests(); err == nil {
		t.Fatalf("an empty table produced %q instead of refusing", got)
	}
}

// TestShellEmitsRefusingFunctionsRatherThanNothing: A BROKEN TABLE MUST EMIT REFUSING FUNCTIONS, NOT NO FUNCTIONS.
func TestShellEmitsRefusingFunctionsRatherThanNothing(t *testing.T) {
	orig := Table
	t.Cleanup(func() { Table = orig })
	Table = nil

	sh := Shell()
	for _, fn := range []string{"authoritative_row()", "gv_all_tests()", "gv_field()"} {
		if !strings.Contains(sh, fn) {
			t.Fatalf("emitted shell does not define %s:\n%s", fn, sh)
		}
	}
	if !strings.Contains(sh, "REFUSING") {
		t.Fatalf("a broken table emitted no refusal:\n%s", sh)
	}

	//nolint:gosec // `sh` is this package's own emitted output, not input
	out, err := exec.CommandContext(t.Context(), "bash", "-c",
		sh+"\nauthoritative_row; echo rc=$?").CombinedOutput()
	if err != nil {
		t.Fatalf("the emitted shell does not parse: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "REFUSING") || !strings.Contains(string(out), "rc=1") {
		t.Fatalf("the emitted refusal did not fire: %s", out)
	}
}

// TestShellRoundTripsThroughBash: The live table's emitted shell must source cleanly and answer the three questions its consumers ask.
func TestShellRoundTripsThroughBash(t *testing.T) {
	sh := Shell()
	//nolint:gosec // ditto
	out, err := exec.CommandContext(t.Context(), "bash", "-c", sh+`
echo "n=${#GATE_VERSIONS[@]}"
a="$(authoritative_row)" || exit 3
echo "auth=$a"
echo "oracle=$(gv_field "$a" 2)"
echo "crash=$(gv_field "$a" 3)"
echo "all=$(gv_all_tests)"
`).CombinedOutput()
	if err != nil {
		t.Fatalf("sourcing the emitted table failed: %v\n%s", err, out)
	}
	row, _ := Authoritative()
	for _, want := range []string{
		"n=" + itoa(len(Table)),
		"oracle=" + row.OracleTest,
		"crash=" + row.CorpusTest,
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("emitted shell did not answer %q:\n%s", want, out)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// TestTSVCarriesEveryRowAndBothTestNames: TSV is what scripts/task_test_refs.py reads.
func TestTSVCarriesEveryRowAndBothTestNames(t *testing.T) {
	lines := strings.Split(strings.TrimRight(TSV(), "\n"), "\n")
	if len(lines) != len(Table) {
		t.Fatalf("TSV has %d rows for a %d-row table", len(lines), len(Table))
	}
	for i, ln := range lines {
		cols := strings.Split(ln, "\t")
		if len(cols) != 4 {
			t.Fatalf("row %d has %d columns, want 4: %q", i, len(cols), ln)
		}
		if cols[1] != Table[i].OracleTest || cols[2] != Table[i].CorpusTest {
			t.Fatalf("row %d does not carry both test names: %q", i, ln)
		}
	}
}
