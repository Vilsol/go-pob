package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestCurseLimitIsMaximumPowerCharges pins the producer for a flag whose CONSUMER was already correct and had nothing to read.
func TestCurseLimitIsMaximumPowerCharges(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	carriers, flagged, matchedOracle, identity := 0, 0, 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(strings.ToLower(strings.Join(strings.Fields(string(raw)), " ")),
			"curse limit is equal to your maximum power charges") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["EnemyCurseLimit"] == 0 {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil {
			continue
		}
		carriers++
		got := env.Player.Output["EnemyCurseLimit"]
		if env.ModDB.Flag(nil, "CurseLimitIsMaximumPowerCharges") {
			flagged++
			if got == env.Player.Output["PowerChargesMax"] {
				identity++
			}
		}
		if math.Abs(got-rec.Output["EnemyCurseLimit"]) <= 0.005*math.Abs(rec.Output["EnemyCurseLimit"]) {
			matchedOracle++
		}
	}

	if carriers < 200 {
		t.Fatalf("REFUSING: only %d carriers of the phrase; the fixture population has moved", carriers)
	}
	const flaggedFloor = 249
	if flagged < flaggedFloor {
		t.Errorf("flag reaches only %d of %d carriers, floor is %d", flagged, carriers, flaggedFloor)
	}
	const identityFloor = 247
	if identity < identityFloor {
		t.Errorf("EnemyCurseLimit == PowerChargesMax on only %d of the %d flagged builds, floor is %d; the consumer is not selecting the branch", identity, flagged, identityFloor)
	}
	if matchedOracle < 265 {
		t.Errorf("oracle agreement REGRESSED to %d of %d", matchedOracle, carriers)
	}
	t.Logf("carriers %d, flagged %d, identity %d, oracle %d", carriers, flagged, identity, matchedOracle)
}
