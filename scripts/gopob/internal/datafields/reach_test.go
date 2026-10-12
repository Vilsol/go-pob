//go:build !js

package datafields

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reachFixture(t *testing.T, variant string) string {
	t.Helper()
	src := filepath.Join("testdata", "reach")
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if strings.HasPrefix(rel, "variants"+string(filepath.Separator)) {
			return nil
		}
		copyTxt(t, path, filepath.Join(dst, strings.TrimSuffix(rel, ".txt")))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if variant != "" {
		name := "variant.go"
		if strings.HasSuffix(variant, "_test") {
			name = "variant_test.go"
		}
		copyTxt(t, filepath.Join(src, "variants", variant+".go.txt"), filepath.Join(dst, "app", name))
	}
	return dst
}

func copyTxt(t *testing.T, from, to string) {
	t.Helper()
	b, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestOnlyReachableDataReadsCount: a go-pob-data getter go-pob never calls leaves the reads alone.
func TestOnlyReachableDataReadsCount(t *testing.T) {
	getterReads := []FieldKey{{"Row", "Hidden"}, {"Row", "Via"}, {"Row", "Iface"}, {"Other", "Name"}}
	for _, tc := range []struct {
		variant string
		want    []FieldKey
	}{
		{"", nil},
		{"direct", []FieldKey{{"Row", "Hidden"}}},
		{"funcvalue", []FieldKey{{"Row", "Via"}}},
		{"iface", []FieldKey{{"Row", "Iface"}}},
		{"reflect", []FieldKey{{"Row", "Hidden"}, {"Row", "Via"}, {"Row", "Iface"}, {"Other", "Name"}}},
		{"reflect_test", nil},
	} {
		t.Run("variant="+tc.variant, func(t *testing.T) {
			res, err := analyze(reachFixture(t, tc.variant), nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := res.Assets["raw/Rows"]; !ok || len(res.Reads[FieldKey{"Row", "ID"}]) == 0 {
				t.Fatalf("the fixture's own load and read are not seen (assets %v); every assertion below would be vacuous", res.Assets)
			}
			for _, k := range getterReads {
				want := false
				for _, w := range tc.want {
					want = want || w == k
				}
				if got := len(res.Reads[k]) > 0; got != want {
					t.Errorf("%s.%s read = %v, want %v (sites %v)", k.Type, k.Field, got, want, res.Reads[k])
				}
			}
		})
	}
}
