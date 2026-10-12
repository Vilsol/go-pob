package data

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const (
	itemToolsCases    = "../testdata/oracle/item_tools_cases.json"
	itemToolsExpected = "../testdata/oracle/item_tools_expected.json"
)

type itemToolsCase struct {
	Name        string   `json:"name"`
	Fn          string   `json:"fn"`
	Line        string   `json:"line"`
	ValueScalar *float64 `json:"valueScalar"`
	Numbers     int      `json:"numbers"`
	Range       float64  `json:"range"`
	Text        string   `json:"text"`
	TextBytes   []int    `json:"textBytes"`
}

func (c itemToolsCase) text() string {
	if c.TextBytes == nil {
		return c.Text
	}
	b := make([]byte, len(c.TextBytes))
	for i, v := range c.TextBytes {
		if v < 0 || v > 255 {
			panic(fmt.Sprintf("TextBytes[%d] = %d is not a byte value", i, v))
		}
		b[i] = byte(v)
	}
	return string(b)
}

type itemToolsResult struct {
	Fn    string  `json:"fn"`
	Name  string  `json:"name"`
	Out   string  `json:"out"`
	Min   string  `json:"min"`
	Max   string  `json:"max"`
	OK    bool    `json:"ok"`
	Error *string `json:"error"`
}

func loadItemToolsFixtures(t *testing.T) ([]itemToolsCase, []itemToolsResult) {
	t.Helper()

	var cases []itemToolsCase
	raw, err := os.ReadFile(itemToolsCases)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))

	var expected []itemToolsResult
	raw, err = os.ReadFile(itemToolsExpected)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &expected))

	testza.AssertEqual(t, len(cases), len(expected), "cases and expected results are out of sync; regenerate the oracle output")

	return cases, expected
}

func TestItemToolsAgainstPoB(t *testing.T) {
	cases, expected := loadItemToolsFixtures(t)

	for i, c := range cases {
		want := expected[i]

		testza.AssertEqual(t, c.Fn, want.Fn, "fixture mismatch at index %d", i)
		testza.AssertNil(t, want.Error, "oracle errored for %q: %v", c.Name, want.Error)

		t.Run(c.Fn+"/"+c.Name, func(t *testing.T) {
			switch c.Fn {
			case "applyValueScalar":
				testza.AssertEqual(t, want.Out, ApplyValueScalar(c.Line, c.ValueScalar, nil, c.Numbers, nil))
			case "applyRange":
				testza.AssertEqual(t, want.Out, ApplyRange(c.Line, c.Range, c.ValueScalar, nil))
			case "getLineRangeMinMax":
				gotMin, gotMax, ok := GetLineRangeMinMax(c.Line)
				testza.AssertEqual(t, want.OK, ok)
				if want.OK {
					testza.AssertEqual(t, want.Min, gotMin)
					testza.AssertEqual(t, want.Max, gotMax)
				}
			case "sanitiseItemText":
				testza.AssertEqual(t, want.Out, SanitiseItemText(c.text()))
			default:
				t.Fatalf("unknown oracle fn %q", c.Fn)
			}
		})
	}
}
