package calculator

import "testing"

// TradeHelpers.lua:199-202: 3.30 renamed Ahuana to Mazcotl; the trade stat id kept the old name.
func TestFindTradeIDOptionMazcotlKeepsAhuanaID(t *testing.T) {
	for _, c := range []struct{ line, id string }{
		{"Bathed in the blood of 1234 sacrificed in the name of Mazcotl", "explicit.pseudo_timeless_jewel_ahuana"},
		{"Bathed in the blood of 1234 sacrificed in the name of Doryani", "explicit.pseudo_timeless_jewel_doryani"},
	} {
		id, value := findTradeIDOption(c.line, "explicit")
		if id != c.id || value != 1234.0 {
			t.Errorf("%q: %q %v, PoB %q 1234", c.line, id, value, c.id)
		}
	}
}
