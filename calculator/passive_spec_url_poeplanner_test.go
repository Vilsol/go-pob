package calculator

import "testing"

// TestPoePlannerVersions330: PoEPlanner's 3.30 version ids resolve as PoB's GameVersions maps them.
func TestPoePlannerVersions330(t *testing.T) {
	for id, want := range map[int]string{55: "3_30", 54: "3_30_ruthless"} {
		got, err := decodeTreeLink(poePlannerLink(id, 0, 0, nil, nil, nil, nil))
		if err != nil {
			t.Fatalf("version %d: %v", id, err)
		}
		if want == "3_30" && string(got.Version) != want {
			t.Fatalf("version %d decoded as %s, want %s", id, got.Version, want)
		}
	}
}
