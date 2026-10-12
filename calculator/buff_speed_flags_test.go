package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

// TestBuffSpeedModsAreFlagged pins upstream 12953654f's rule for Adrenaline.
func TestBuffSpeedModsAreFlagged(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	if raw.LatestVersion != "3.20" {
		t.Skipf("SKIP: this test pins 3.20 behaviour but the process loaded %s game data", raw.LatestVersion)
	}

	const (
		build       = "0016-berserker-vaal-reave.xml"
		wantAttack  = 238.0
		wantAilment = 32.0
	)

	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

	if !env.ModDB.Flag(nil, "Adrenaline") && !env.ModDB.Flag(nil, "HerEmbrace") {
		t.Fatalf("REFUSING: %s sets neither Adrenaline nor HerEmbrace, so it cannot see this "+
			"defect — wrong fixture", build)
	}

	attackCfg := &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagAttack)}
	ailmentCfg := &moddb.ListCfg{Flags: utils.Ptr(mod.MFlagDot | mod.MFlagAilment)}

	if got := env.ModDB.Sum(mod.TypeIncrease, ailmentCfg, "Speed"); got != wantAilment {
		t.Errorf("%s: Speed INC on a Dot|Ailment cfg = %g, want %g. An unflagged buff mod matches "+
			"every cfg, so it leaks into passes the game never applies it to.", build, got, wantAilment)
	}
	if got := env.ModDB.Sum(mod.TypeIncrease, attackCfg, "Speed"); got != wantAttack {
		t.Errorf("%s: Speed INC on an Attack cfg = %g, want %g — the buff must still REACH the "+
			"attack pass; scoping it is not the same as removing it.", build, got, wantAttack)
	}
}
