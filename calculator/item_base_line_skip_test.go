package calculator

import (
	"context"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestBaseLineIsNotAModLine: Item.lua:1051-1052 -- `-- Base lines don't need mod parsing, skip it` / `goto continue`.
func TestBaseLineIsNotAModLine(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(context.Background()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	const raw = `Rarity: UNIQUE
Gruthkul's Pelt
Wyrmscale Doublet
Armour: 1416
ArmourBasePercentile: 1
Evasion: 1416
EvasionBasePercentile: 1
Quality: 20
LevelReq: 38
Implicits: 1
{tags:resource,life}{range:0.5}(4-6)% increased maximum Life
100% increased Global Physical Damage
+(240-300) to maximum Life`

	i := ParseItem(raw)

	if i.BaseName != "Wyrmscale Doublet" {
		t.Fatalf("REFUSING: base did not resolve (%q) -- the rest of this test would be vacuous", i.BaseName)
	}
	if len(i.ExplicitModLines) == 0 {
		t.Fatal("REFUSING: no explicit lines parsed at all")
	}

	for n, ml := range i.ExplicitModLines {
		if strings.HasPrefix(ml.Line, "Wyrmscale Doublet") {
			t.Errorf("explicit[%d] is the BASE line: %q (extra %q)", n, ml.Line, ml.Extra)
		}
	}
	if got := i.ExplicitModLines[0].Line; !strings.Contains(got, "increased Global Physical Damage") {
		t.Errorf("explicit[0]: got %q, want the first real modifier", got)
	}
}
