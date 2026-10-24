package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

var esLeechOnFullSoulDrinker = regexp.MustCompile(`(^|[^0-9])48999([^0-9]|$)`)

var _ = registerCorpusCheck("TestEnergyShieldLeechOnFullReachesTheCondition", func() corpusCheck {
	dir, err := corpusDirForVersion(raw.LatestVersion)
	return &esLeechOnFullCheck{dir: dir, dirErr: err}
})

type esLeechOnFullCheck struct {
	mu                                   sync.Mutex
	dir                                  string
	dirErr                               error
	carriers, fromItemText, conditionSet int
}

func (c *esLeechOnFullCheck) visit(name string, _ oracleRecord, _ bool, env *Environment) {
	const itemPhrase = "not removed at Full Energy Shield"
	if c.dirErr != nil || env == nil || env.ModDB == nil {
		return
	}
	rawXML, err := os.ReadFile(filepath.Join(c.dir, name))
	if err != nil {
		return
	}
	hasItemPhrase := strings.Contains(string(rawXML), itemPhrase)
	if !hasItemPhrase && !esLeechOnFullSoulDrinker.Match(rawXML) {
		return
	}
	if !env.ModDB.Flag(nil, "CanLeechEnergyShieldOnFullEnergyShield") {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.carriers++
	if hasItemPhrase {
		c.fromItemText++
	}
	if env.ModDB.Conditions["Leeching"] && env.ModDB.Conditions["LeechingEnergyShield"] {
		c.conditionSet++
	}
}

// TestEnergyShieldLeechOnFullReachesTheCondition pins ModParser.lua:3000 and :3013 together with their consumer at CalcPerform.lua:877-880.
func TestEnergyShieldLeechOnFullReachesTheCondition(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	c := corpusWalkCheck[*esLeechOnFullCheck](t)
	carriers, fromItemText, conditionSet := c.carriers, c.fromItemText, c.conditionSet

	if carriers < 130 {
		t.Fatalf("REFUSING: only %d builds carry CanLeechEnergyShieldOnFullEnergyShield; the row or the population has moved", carriers)
	}
	if conditionSet != carriers {
		t.Errorf("CalcPerform.lua:877-880 did not fire on every carrier: %d of %d set Leeching+LeechingEnergyShield", conditionSet, carriers)
	}
	if fromItemText < 100 {
		t.Errorf("ModParser.lua:3000's row REGRESSED: only %d of %d carriers carry its item phrase", fromItemText, carriers)
	}
	if carriers-fromItemText < 20 {
		t.Errorf("the NON-item-text carriers REGRESSED to %d: ModParser.lua:3013's tree-sourced row is the only thing that produces them", carriers-fromItemText)
	}
}
