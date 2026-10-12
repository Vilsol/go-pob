package data

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data/raw"
)

const (
	modScalabilityCases    = "../testdata/oracle/mod_scalability_cases.json"
	modScalabilityExpected = "../testdata/oracle/mod_scalability_expected.json"
)

type modScalabilityCase struct {
	Line            string   `json:"line"`
	Range           float64  `json:"range"`
	ValueScalar     *float64 `json:"valueScalar"`
	BaseValueScalar *float64 `json:"baseValueScalar"`
}

func TestApplyRangeScalabilityAgainstPoB(t *testing.T) {
	var cases []modScalabilityCase
	readJSONFixture(t, modScalabilityCases, &cases)

	var expected struct {
		PoBVersion string   `json:"pobVersion"`
		Results    []string `json:"results"`
	}
	readJSONFixture(t, modScalabilityExpected, &expected)

	testza.AssertEqual(t, len(cases), len(expected.Results),
		"the case and expectation fixtures have different lengths — re-record with "+
			"`task oracle-mod-scalability`")
	testza.AssertTrue(t, len(cases) > 100,
		"only %d cases; the fixture has been truncated and this test grades almost nothing",
		len(cases))

	testza.AssertTrue(t, len(raw.ModScalability) > 10000,
		"raw.ModScalability holds %d rows; the generated table is empty or truncated, so every "+
			"lookup below misses and the legacy fallback is what is being graded",
		len(raw.ModScalability))

	for i, c := range cases {
		name := c.Line
		if len(name) > 60 {
			name = name[:60]
		}
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			testza.AssertEqual(t, expected.Results[i],
				ApplyRange(c.Line, c.Range, c.ValueScalar, c.BaseValueScalar))
		})
	}
}

func readJSONFixture(t *testing.T, path string, into any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is unreadable: %v\n  Re-record it with `task oracle-mod-scalability`.", path, err)
	}
	if err := json.Unmarshal(body, into); err != nil {
		t.Fatalf("%s is not the JSON this test expects: %v", path, err)
	}
}
