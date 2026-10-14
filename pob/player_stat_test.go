package pob

import (
	"encoding/xml"
	"testing"
)

// TestPlayerStatLoadsANonNumericValue: Build.lua:1039 writes tostring(statVal), so a boolean output saves as "true"/"false"; PoB never reads it back.
func TestPlayerStatLoadsANonNumericValue(t *testing.T) {
	var b struct {
		Stats []PlayerStat `xml:"PlayerStat"`
	}
	in := `<Build><PlayerStat stat="Life" value="1234"/><PlayerStat stat="ChaosInoculation" value="false"/></Build>`
	if err := xml.Unmarshal([]byte(in), &b); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(b.Stats) != 2 || b.Stats[0].Value != 1234 || b.Stats[1].Stat != "ChaosInoculation" {
		t.Fatalf("got %+v", b.Stats)
	}
}
