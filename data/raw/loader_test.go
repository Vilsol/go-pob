package raw

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
	raw2 "github.com/Vilsol/go-pob-data/raw"
)

func TestRawLoader(t *testing.T) {
	d, err := LoadRaw[*raw2.GrantedEffectStatSet](LatestVersion, "GrantedEffectStatSets", nil)
	testza.AssertNoError(t, err)

	marshal, _ := json.Marshal(d)
	_ = os.WriteFile("out.json", marshal, 0755)
}
