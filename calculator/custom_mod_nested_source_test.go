package calculator

import (
	"testing"
)

// TestCustomModifierNestedModCarriesTheBlockSource: ConfigTab.lua:1121 and :1140 use modLib.setSource, which sources an ExtraAura's inner mod too.
func TestCustomModifierNestedModCarriesTheBlockSource(t *testing.T) {
	assertNamedRows(t, "0001-necromancer-unknown.xml", func(name string) bool { return name == "Default" })
}
