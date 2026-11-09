package exposition

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func TestTreeFunctionsRefuseAnUnknownVersion(t *testing.T) {
	if _, err := treeVersion(data.LatestTreeVersion); err != nil {
		t.Fatalf("the latest tree version is refused: %v", err)
	}

	const unknown = pob.TreeVersion("3_25")
	calls := map[string]func() error{
		"GetRawTree": func() error { _, err := GetRawTree(unknown); return err },
		"CalculateAllocationPaths": func() error {
			_, err := CalculateAllocationPaths(unknown, nil, []int64{1})
			return err
		},
		"CalculatePrunableNodes": func() error {
			_, err := CalculatePrunableNodes(unknown, nil, []int64{1})
			return err
		},
		"CalculateDependentNodes": func() error {
			_, err := CalculateDependentNodes(unknown, nil, 1, []int64{1})
			return err
		},
		"CalculateDependentNodesWith": func() error {
			_, err := CalculateDependentNodesWith(unknown, nil, 1, []int64{1}, nil)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s(%q) panicked: %v", name, unknown, r)
				}
			}()
			if err := call(); err == nil || !strings.Contains(err.Error(), string(unknown)) {
				t.Fatalf("%s(%q) returned %v, want an error naming the version", name, unknown, err)
			}
		})
	}
}
