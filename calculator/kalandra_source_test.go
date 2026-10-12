package calculator

import (
	"strings"
	"testing"
)

// TestKalandraMirroredModsCarryTheRingSource: CalcSetup.lua:1098-1100 copy each mirrored mod and source it with Kalandra's Touch.
func TestKalandraMirroredModsCarryTheRingSource(t *testing.T) {
	assertNamedRows(t, "13276-ladder-cuba_5717.cubamrc.xml", func(name string) bool { return strings.Contains(name, "Kalandra's Touch") })
}
