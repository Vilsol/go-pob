package calculator

import (
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const modParseExpected = "../testdata/oracle/mod_parse_expected.json"

func fixturePath(env, def string) string {
	if p := os.Getenv(env); p != "" {
		return p
	}
	return def
}

type modParseCase struct {
	Line   string  `json:"line"`
	Parsed bool    `json:"parsed"`
	Extra  *string `json:"extra"`
	Count  int     `json:"count"`
}

func goParsed(line string) bool {
	entry := ParseMod(line, false)
	return entry.ModList != nil && strings.TrimSpace(entry.Extra) == ""
}

const knownModParserGaps = 3

var modGapCount atomic.Int64

func TestModParserAgainstPoB(t *testing.T) {
	var cases []modParseCase
	raw, err := os.ReadFile(fixturePath("MOD_PARSE_EXPECTED", modParseExpected))
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(raw, &cases))
	testza.AssertGreater(t, len(cases), 0)

	pobFull := func(c modParseCase) bool {
		return c.Parsed && (c.Extra == nil || strings.TrimSpace(*c.Extra) == "")
	}

	var gaps []string
	total := 0
	for _, c := range cases {
		if !pobFull(c) {
			continue
		}
		total++
		if !goParsed(c.Line) {
			gaps = append(gaps, c.Line)
		}
	}

	modGapCount.Store(int64(len(gaps)))
	t.Logf("%d/%d lines PoB fully parses are parsed by go-pob; %d gaps remain",
		total-len(gaps), total, len(gaps))
	for _, g := range gaps {
		t.Logf("  GAP: %q", g)
	}

	testza.AssertLessOrEqual(t, len(gaps), knownModParserGaps,
		"more ModParser gaps than the ratchet allows; a mod that used to parse no longer does")

	if len(gaps) < knownModParserGaps {
		t.Logf("ModParser improved: lower knownModParserGaps from %d to %d", knownModParserGaps, len(gaps))
	}
}
