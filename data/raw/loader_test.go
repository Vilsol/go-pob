package raw

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestRawLoader(t *testing.T) {
	d, err := LoadRaw[[]*GrantedEffectStatSet](LatestVersion, "GrantedEffectStatSets")
	testza.AssertNoError(t, err)

	marshal, _ := json.Marshal(d)
	_ = os.WriteFile("out.json", marshal, 0755)
}
