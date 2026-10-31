//go:build !js

package calculator2

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// TestPoE2DoesNotDependOnPoE1Data keeps PoE1's data packages, and the tables their inits load, out of the PoE2 binary.
func TestPoE2DoesNotDependOnPoE1Data(t *testing.T) {
	forbidden := []string{"github.com/Vilsol/go-pob/data/raw", "github.com/Vilsol/go-pob/data"}
	for _, tc := range []struct {
		pkg string
		env []string
	}{
		{"github.com/Vilsol/go-pob/calculator2", nil},
		{"github.com/Vilsol/go-pob/calculator2", []string{"GOOS=js", "GOARCH=wasm"}},
		{"github.com/Vilsol/go-pob/wasm2", []string{"GOOS=js", "GOARCH=wasm"}},
	} {
		cmd := exec.CommandContext(t.Context(), "go", "list", "-deps", tc.pkg) //nolint:gosec // package paths from the table above
		cmd.Env = append(os.Environ(), tc.env...)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("go list -deps %s %v: %v", tc.pkg, tc.env, err)
		}
		deps := strings.Fields(string(out))
		if !slices.Contains(deps, "github.com/Vilsol/go-pob/moddb") {
			t.Fatalf("go list -deps %s %v does not list moddb, so it is not seeing the import graph", tc.pkg, tc.env)
		}
		for _, f := range forbidden {
			if slices.Contains(deps, f) {
				t.Errorf("%s %v depends on %s", tc.pkg, tc.env, f)
			}
		}
	}
}
