package mcp

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAuditCustomModOffence: a custom damage line is audited with the main skill's DPS with and without it, and the
// figure is what removing the line with an edit gives.
func TestAuditCustomModOffence(t *testing.T) {
	s := testServer(t)
	res := call(t, s, "import_build", map[string]any{"source": fixtureWithConfig(t, `<CustomModifierBlock title="Default" enabled="true">100% more Damage</CustomModifierBlock>`)})
	h := res["handle"].(string)
	var row map[string]any
	for _, r := range asList(res["audit"].(map[string]any)["custom_offence"]) {
		if m := r.(map[string]any); m["stat"] == mainDPS {
			row = m
		}
	}
	if row == nil {
		t.Fatalf("audit has no MainDPS custom offence row: %v", res["audit"])
	}
	with, without := row["with"].(float64), row["without"].(float64)
	if r := with / without; r < 1.9 || r > 2.1 {
		t.Fatalf("100%% more Damage: MainDPS %v with, %v without", with, without)
	}
	if !strings.Contains(fmt.Sprint(row["set"]), `"Default" (active)`) {
		t.Fatalf("the row does not name its config set: %v", row)
	}
	edit := call(t, s, "edit_build", map[string]any{"handle": h, "dry_run": true, "watch": []string{mainDPS},
		"ops": ops(map[string]any{"op": "unset_config", "var": "customMods"})})
	if d := deltaOf(edit, mainDPS); math.Abs(d-(without-with)) > 0.002*with {
		t.Fatalf("removing the custom line moves MainDPS by %v; the audit says %v", d, without-with)
	}
}

// TestAuditEveryConfigSet: config values above their limits are found in every config set, each line naming its set,
// and a stored Rage with no Rage source in the build is questioned.
func TestAuditEveryConfigSet(t *testing.T) {
	s := testServer(t)
	path := fixtureWithConfig(t, "set:"+`</ConfigSet><ConfigSet id="6" title="Bossing"><Input number="4444" name="multiplierRage"/>`)
	audit := fmt.Sprint(call(t, s, "import_build", map[string]any{"source": path})["audit"])
	for _, want := range []string{`config set "Mapping" (active): Config multiplierRage is 120`, `config set "Bossing": Config multiplierRage is 4444`} {
		if !strings.Contains(audit, want) {
			t.Fatalf("audit lacks %q: %s", want, audit)
		}
	}

	raw, err := os.ReadFile(fixture("1.xml"))
	if err != nil {
		t.Fatal(err)
	}
	noSource := filepath.Join(t.TempDir(), "rage.xml")
	if err := os.WriteFile(noSource, []byte(strings.Replace(string(raw), "\t</Config>", `<Input number="444" name="multiplierRage"/>`+"\n\t</Config>", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	audit = fmt.Sprint(call(t, s, "import_build", map[string]any{"source": noSource})["audit"])
	if !strings.Contains(audit, "multiplierRage is 444, but no Rage source found") {
		t.Fatalf("Rage with no source is not questioned: %s", audit)
	}
}

// TestAuditLegacyCustomModsIgnored: a set's legacy customMods text that PoB drops because the set has custom modifier
// blocks is named.
func TestAuditLegacyCustomModsIgnored(t *testing.T) {
	s := testServer(t)
	path := fixtureWithConfig(t, `<Input string="+1000 to maximum Life" name="customMods"/><CustomModifierBlock title="Default" enabled="true">+10% to Fire Resistance</CustomModifierBlock>`)
	audit := fmt.Sprint(call(t, s, "import_build", map[string]any{"source": path})["audit"])
	if !strings.Contains(audit, `the legacy customMods text "+1000 to maximum Life" is ignored`) {
		t.Fatalf("audit = %s", audit)
	}
}
