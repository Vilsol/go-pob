package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestConditionalKeystoneGrants pins ModParser.lua:4694 and :5242, two rows that grant a whole KEYSTONE behind a charges-at-maximum threshold.
func TestConditionalKeystoneGrants(t *testing.T) {
	for _, tc := range []struct {
		line          string
		keystone      string
		stat          string
		thresholdStat string
	}{
		{"You have Iron Reflexes while at maximum Frenzy Charges", "Iron Reflexes", "FrenzyCharges", "FrenzyChargesMax"},
		{"You have Mind over Matter while at maximum Power Charges", "Mind Over Matter", "PowerCharges", "PowerChargesMax"},
	} {
		t.Run(tc.keystone, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v mods, want 1", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			m := e.ModList[0]
			if m.Name() != "Keystone" {
				t.Fatalf("mod name = %q, want Keystone", m.Name())
			}
			if got, ok := m.Value().List().(string); !ok || got != tc.keystone {
				t.Errorf("keystone = %v, want %q", m.Value().List(), tc.keystone)
			}
			var st *mod.StatThresholdTag
			for _, tg := range m.Tags() {
				if s, ok := tg.(*mod.StatThresholdTag); ok {
					st = s
				}
			}
			if st == nil {
				t.Fatalf("no StatThresholdTag; tags = %v", m.Tags())
			}
			if st.Stat != tc.stat {
				t.Errorf("stat = %q, want %q", st.Stat, tc.stat)
			}
			if st.TagThresholdStat != tc.thresholdStat {
				t.Errorf("thresholdStat = %q, want %q", st.TagThresholdStat, tc.thresholdStat)
			}
			if st.Threshold != 0 {
				t.Errorf("a literal threshold %v is set; upstream passes thresholdStat only", st.Threshold)
			}
		})
	}
}
