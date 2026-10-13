package calculator

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestConfigFlipFixtures replays the committed flip variants.
func TestConfigFlipFixtures(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "config-flip", "expected.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	recs := make([]configFlipFixtureRecord, 0, strings.Count(string(raw), "\n")+1)
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var r configFlipFixtureRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal(err)
		}
		recs = append(recs, r)
	}
	if len(recs) == 0 {
		t.Fatal("REFUSING: no fixture records")
	}
	corpusDir := corpusDirForLoadedData(t)
	guardPicks, pairsPicks := loadGuardPicks(t), loadPairsPicks(t)
	cache := NewEnvironmentCache()
	for _, r := range recs {
		t.Run(r.Var+"="+r.Value+"/"+strings.TrimSuffix(r.Carrier, ".xml"), func(t *testing.T) {
			base, err := os.ReadFile(filepath.Join(corpusDir, r.Carrier))
			if err != nil {
				t.Fatal(err)
			}
			v, err := flipVariantXML(base, r.Var, r.Attr, r.Value)
			if err == nil {
				err = verifyFlipVariant(base, v, r.Var, r.Value, r.Attr == "")
			}
			if err != nil {
				t.Fatal(err)
			}
			ours := evalFlipOurs(v, guardPicks[r.Carrier], pairsPicks[r.Carrier], cache)
			if ours.err != "" {
				t.Fatal(ours.err)
			}
			wrong := 0
			for _, key := range slices.Sorted(maps.Keys(r.PoB)) {
				prefix, name := "", key
				for _, p := range []string{"Minion.MainHand.", "Minion.OffHand.", "MainHand.", "OffHand.", "Enemy.", "Minion."} {
					if strings.HasPrefix(key, p) {
						prefix, name = p, strings.TrimPrefix(key, p)
						break
					}
				}
				got := ours.tables[prefix][name]
				if !oracleClose(got, r.PoB[key], oracleDumpTolerance) {
					wrong++
					if wrong <= 5 {
						t.Errorf("%s: ours %.6g, PoB %.6g", key, got, r.PoB[key])
					}
				}
			}
			if wrong > 5 {
				t.Errorf("... %d of %d moved stats wrong", wrong, len(r.PoB))
			}
		})
	}
}
