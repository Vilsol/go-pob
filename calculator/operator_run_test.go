package calculator

import (
	"os"
	"testing"
)

func requireOperatorRun(t *testing.T, why string) {
	t.Helper()
	if os.Getenv("OPERATOR") != "true" {
		t.Skipf("SKIP: operator-side only (%s). Set OPERATOR=true, or use the task that runs it.", why)
	}
}
