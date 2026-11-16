package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// TestOptimizeFreesPointsAndRefills: at full points, Blood Magic (three points on its path) beats three 1-point Life
// nodes only once three of them are dropped together; one-change moves never get there, a free-and-refill move does.
func TestOptimizeFreesPointsAndRefills(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	args := swapSpace(h, []string{"29933", "40362", "58271", "65034", "65167", "57279"}, 600)
	res := call(t, s, "optimize", args)
	if m := res["space"].(map[string]any)["method"]; m != "coordinate_ascent" {
		t.Fatalf("method %v", m)
	}
	top := rows(res)[0]
	if c := fmt.Sprint(top["choices"]); top["feasible"] != true || !strings.Contains(c, "57279") {
		t.Fatalf("best result without Blood Magic after %v: %v", res["evaluations"], top)
	}
}
