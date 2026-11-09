package calculator2

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

type partyOracleSide struct {
	Player map[string]json.Number `json:"player"`
	Enemy  map[string]json.Number `json:"enemy"`
}

type partyOracle struct {
	Exported    map[string]string `json:"exported"`
	ExportOrder []string          `json:"exportOrder"`
	Receivers   []struct {
		Build   string          `json:"build"`
		Without partyOracleSide `json:"without"`
		With    partyOracleSide `json:"with"`
	} `json:"receivers"`
}

func loadPartyOracle(t *testing.T, name string) partyOracle {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "oracle", "poe2_party_"+name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var o partyOracle
	if err := dec.Decode(&o); err != nil {
		t.Fatal(err)
	}
	if len(o.Receivers) == 0 || len(o.ExportOrder) == 0 {
		t.Fatalf("%s: the oracle holds no receiver or no export", name)
	}
	return o
}

func partyOracleNumber(n json.Number) float64 {
	v, err := strconv.ParseFloat(string(n), 64)
	if err != nil {
		return math.NaN()
	}
	return v
}

func partyReceiverOutputs(t *testing.T, xmlName string, buffers []PartyBuffer) (map[string]float64, map[string]float64) {
	t.Helper()
	xml, err := os.ReadFile(filepath.Join("testdata", xmlName))
	if os.IsNotExist(err) {
		xml, err = os.ReadFile(filepath.Join("..", "wasm2", "exposition", "testdata", xmlName))
	}
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range buffers {
		name := b.Name
		build.Party.ImportedBuffs = append(build.Party.ImportedBuffs, pob2.PartyBuffs{Name: &name, Text: b.Text})
	}
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	if err != nil {
		t.Fatal(err)
	}
	return env.Player.Output, env.Enemy.Output
}

// TestPartyImportVsPoBPoE2 imports a party export into receiver builds and grades every output PoB-PoE2's party
// tab changes (oracle/poe2_party.lua): the giver fixture is a build's own export, the buffers fixture a hand-written
// set covering each buffer family. A stat our calculation already disagrees on without a party is reported, not graded.
func TestPartyImportVsPoBPoE2(t *testing.T) {
	for _, fixture := range []string{"giver", "buffers"} {
		o := loadPartyOracle(t, fixture)
		var buffers []PartyBuffer
		for _, name := range o.ExportOrder {
			buffers = append(buffers, PartyBuffer{Name: name, Text: o.Exported[name]})
		}
		for _, r := range o.Receivers {
			basePlayer, baseEnemy := partyReceiverOutputs(t, r.Build, nil)
			player, enemy := partyReceiverOutputs(t, r.Build, buffers)
			for _, side := range []struct {
				name           string
				without, with  map[string]json.Number
				ourBase, ourAt map[string]float64
			}{
				{"player", r.Without.Player, r.With.Player, basePlayer, player},
				{"enemy", r.Without.Enemy, r.With.Enemy, baseEnemy, enemy},
			} {
				keys := map[string]bool{}
				for k := range side.with {
					keys[k] = true
				}
				for k := range side.ourAt {
					keys[k] = true
				}
				names := make([]string, 0, len(keys))
				for k := range keys {
					names = append(names, k)
				}
				sort.Strings(names)
				moved, matched, ungraded := 0, 0, 0
				for _, k := range names {
					wantBase, wantAt := partyOracleNumber(side.without[k]), partyOracleNumber(side.with[k])
					gotBase, gotAt := side.ourBase[k], side.ourAt[k]
					if side.without[k] == "" {
						wantBase = 0
					}
					if side.with[k] == "" {
						wantAt = 0
					}
					if wantBase == wantAt && gotBase == gotAt {
						continue
					}
					moved++
					if !poe2OracleClose(gotBase, wantBase, poe2OracleTolerance) {
						ungraded++
						t.Logf("%s %s %s %s: ungraded, ours %v vs PoB %v without the party", fixture, r.Build, side.name, k, gotBase, wantBase)
						continue
					}
					if poe2OracleClose(gotAt, wantAt, poe2OracleTolerance) {
						matched++
						continue
					}
					t.Errorf("%s %s %s %s: ours %v -> %v, PoB %v -> %v", fixture, r.Build, side.name, k, gotBase, gotAt, wantBase, wantAt)
				}
				t.Logf("%s %s %s: %d outputs moved, %d matched, %d ungraded", fixture, r.Build, side.name, moved, matched, ungraded)
				if side.name == "player" && matched == 0 {
					t.Errorf("%s %s: no moved player output matched", fixture, r.Build)
				}
			}
		}
	}
}
