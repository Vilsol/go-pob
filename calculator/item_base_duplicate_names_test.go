package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestDuplicateItemBaseNamesMatchPoBWinner: PoB's src/Data/Bases/*.lua declares 954 `itemBases["…"]` lines resolving to 937 distinct names.
func TestDuplicateItemBaseNamesMatchPoBWinner(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	names := []string{
		"Two-Stone Ring", "Avian Twins Talisman", "Monkey Paw Talisman",
		"Blunt Arrow Quiver", "Broadhead Arrow Quiver", "Fire Arrow Quiver",
		"Penetrating Arrow Quiver", "Serrated Arrow Quiver", "Sharktooth Arrow Quiver",
		"Spike-Point Arrow Quiver", "Two-Point Arrow Quiver",
	}
	wantType := map[string]string{
		"Two-Stone Ring":       "Ring",
		"Avian Twins Talisman": "Amulet",
		"Monkey Paw Talisman":  "Amulet",
	}

	for _, name := range names {
		base := ItemBaseByName(name)
		if base == nil {
			t.Errorf("%q: no base — PoB ships one", name)
			continue
		}
		typ, ok := wantType[name]
		if !ok {
			typ = "Quiver"
		}
		if base.Type != typ {
			t.Errorf("%q: Type = %q, want %q", name, base.Type, typ)
		}
		if base.Req.Str != 0 || base.Req.Dex != 0 || base.Req.Int != 0 {
			t.Errorf("%q: Req attrs = %d/%d/%d, want 0/0/0", name, base.Req.Str, base.Req.Dex, base.Req.Int)
		}
		if base.SocketLimit != 0 {
			t.Errorf("%q: SocketLimit = %d, want 0", name, base.SocketLimit)
		}
		if base.Armour != nil {
			t.Errorf("%q: Armour = %+v, want nil", name, base.Armour)
		}
		if base.Weapon != nil {
			t.Errorf("%q: Weapon = %+v, want nil", name, base.Weapon)
		}
	}
}
