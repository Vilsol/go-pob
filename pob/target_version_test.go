package pob

import "testing"

// TestNeedsVersionConversion: Build.lua:99-106 opens the conversion popup for any target version but the live one; Convert makes it live.
func TestNeedsVersionConversion(t *testing.T) {
	for _, v := range []GameVersion{"", "2_6"} {
		b := &PathOfBuilding{Build: Build{TargetVersion: v}}
		if !b.NeedsVersionConversion() {
			t.Errorf("%q: no conversion", v)
		}
		b.ConvertToLiveVersion()
		if b.NeedsVersionConversion() || b.Build.TargetVersion != LiveTargetVersion {
			t.Errorf("%q: converted to %q", v, b.Build.TargetVersion)
		}
	}
	if (&PathOfBuilding{Build: Build{TargetVersion: LiveTargetVersion}}).NeedsVersionConversion() {
		t.Error("the live version asks to convert")
	}
}
