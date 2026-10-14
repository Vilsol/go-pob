package calculator

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func requireOperatorRun(t *testing.T, why string) {
	t.Helper()
	if os.Getenv("OPERATOR") != "true" {
		t.Skipf("SKIP: operator-side only (%s). Set OPERATOR=true, or use the task that runs it.", why)
	}
}

func requireTaskInput(t *testing.T, task string, keys ...string) []string {
	t.Helper()
	vals := make([]string, len(keys))
	var missing []string
	for i, k := range keys {
		vals[i] = os.Getenv(k)
		if vals[i] == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) == len(keys) {
		t.Skipf("SKIP: no input (%s unset); run this through `task %s`", strings.Join(keys, ", "), task)
	}
	if len(missing) > 0 {
		t.Fatalf("REFUSING: %s unset while %s is set; run this through `task %s`",
			strings.Join(missing, ", "), strings.Join(slices.DeleteFunc(slices.Clone(keys), func(k string) bool {
				return slices.Contains(missing, k)
			}), ", "), task)
	}
	return vals
}
