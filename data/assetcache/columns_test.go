package assetcache

import (
	"encoding/json"
	"reflect"
	"testing"
)

type pruneRow struct {
	ID    string            `json:"Id"`
	N     int               `json:"N"`
	P     *int              `json:"P"`
	B     bool              `json:"B"`
	L     []int             `json:"L"`
	O     map[string]string `json:"O"`
	F     float64           `json:"F"`
	Plain int
}

func TestPruneJSONKeepsWhatTheRowDecodes(t *testing.T) {
	raw := []byte(` [ {"Id":"a,]}","N":0,"P":0,"B":false,"L":[],"O":{"k":"]"},"F":0.0,"Plain":3,"Drop":[1,{"x":"}"}],"N\u0041":1} ,
	{"id":"","n":7,"P":null,"L":null,"Other":"\"","F":-0} ]`)
	cols := tableColumns([]reflect.Type{reflect.TypeFor[pruneRow]()})
	pruned, err := pruneJSON(raw, cols)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"Id":"a,]}","P":0,"L":[],"O":{"k":"]"},"F":0.0,"Plain":3,"N\u0041":1},{"n":7,"F":-0}]`
	if string(pruned) != want {
		t.Errorf("pruned\n%s\nwant\n%s", pruned, want)
	}
	var got, orig []pruneRow
	if err := json.Unmarshal(pruned, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &orig); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("pruned rows %+v, original %+v", got, orig)
	}
}

func TestPruneJSONRefusesWhatIsNotATable(t *testing.T) {
	cols := tableColumns([]reflect.Type{reflect.TypeFor[pruneRow]()})
	for _, raw := range []string{`{}`, `[1]`, `[{"Id":"a"}`, `[{"Id":"a"}] x`, `[{"Id":}]`, `[{"Id":"a}]`} {
		if _, err := pruneJSON([]byte(raw), cols); err == nil {
			t.Errorf("%s pruned without an error", raw)
		}
	}
	if pruned, err := pruneJSON([]byte(` [ ] `), cols); err != nil || string(pruned) != "[]" {
		t.Errorf("an empty table pruned to %q, %v", pruned, err)
	}
}

func TestColumnsFingerprintFollowsTheRowType(t *testing.T) {
	type other struct {
		ID string `json:"Id"`
	}
	a := columnsFingerprint([]reflect.Type{reflect.TypeFor[pruneRow]()})
	if a != columnsFingerprint([]reflect.Type{reflect.TypeFor[pruneRow]()}) {
		t.Error("the fingerprint is not stable")
	}
	if a == columnsFingerprint([]reflect.Type{reflect.TypeFor[other]()}) {
		t.Error("two row types share a fingerprint")
	}
}
