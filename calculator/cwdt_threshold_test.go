package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestCWDTThreshold: CalcTriggers.lua:1177-1178 is the whole of it.
//
//	local thresholdMod = calcLib.mod(env.player.mainSkill.skillModList, nil, "CWDTThreshold")
func TestCWDTThreshold(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true required")
	}
	cases := []struct {
		build string
		want  float64
	}{
		{"0131-elementalist-vaal-molten-shell", 501.6},
		{"0156-assassin-steelskin", 2874},
		{"0277-reliquarian-blade-blast-2", 996.3},
		{"0791-elementalist-immortal-call", 2944.8},
		{"0829-assassin-essence-drain-2", 725},
		{"1025-hierophant-punishment", 725},
		{"1562-luminary-steelskin", 897},
	}
	for _, tc := range cases {
		t.Run(tc.build, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tc.build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := builds.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)
			got, ok := env.Player.Output["CWDTThreshold"]
			if !ok {
				t.Fatalf("we never write CWDTThreshold; PoB has %g", tc.want)
			}
			if !oracleClose(got, tc.want, oracleDumpTolerance) {
				t.Errorf("CWDTThreshold = %g, PoB %g", got, tc.want)
			}
		})
	}
}

// TestCWDTThresholdAbsentWithoutTheGem: A build with no CWDT gem must not grow the key.
func TestCWDTThresholdAbsentWithoutTheGem(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true required")
	}
	const build = "0001-necromancer-unknown"
	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)
	if v, ok := env.Player.Output["CWDTThreshold"]; ok {
		t.Errorf("CWDTThreshold = %g on a build PoB writes no CWDTThreshold for — that is an EXTRA key", v)
	}
}
