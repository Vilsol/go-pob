package calculator

import (
	"strings"
	"testing"
)

// TestEmbraceMadnessActiveConfig pins ConfigOptions.lua:423-425.
func TestEmbraceMadnessActiveConfig(t *testing.T) {
	guard := func(t *testing.T, xml string) {
		if !strings.Contains(xml, `name="embraceMadnessActive"`) {
			t.Fatalf("PREMISE BROKEN: build does not tick embraceMadnessActive")
		}
	}
	assertBuildStats(t, "0650-chieftain-lancing-steel-of-spraying.xml", []string{"FireResistTotal", "IgniteDPS"}, guard)
}
