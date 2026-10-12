package calculator

import (
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func linearLongestItemBase(name string) (string, string, string) {
	baseName, prefix, suffix := "", "", ""
	best := -1
	for _, itemBaseName := range ItemBaseNames() {
		if itemBaseName == "" {
			continue
		}
		s := strings.Index(name, itemBaseName)
		if s < 0 {
			continue
		}
		if l := len(itemBaseName) - 1; l > best {
			best = l
			prefix = name[:s]
			suffix = name[s+len(itemBaseName):]
			baseName = itemBaseName
		}
	}
	return baseName, prefix, suffix
}

func indexItemBaseInName(name string) (string, string, string) {
	if b, s := findItemBaseInName(name); s >= 0 {
		return b, name[:s], name[s+len(b):]
	}
	if hasEmptyItemBaseName() {
		return "", "", name
	}
	return "", "", ""
}

// TestFindItemBaseInNameMatchesLinearScan: The bigram index is a pure prefilter.
func TestFindItemBaseInNameMatchesLinearScan(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	names := ItemBaseNames()
	testza.AssertGreater(t, len(names), 0)

	inputs := []string{"", "a", "ab", "Nothing Matches Here", "Two-Toned Boots"}
	for i := 0; i < len(names); i += 41 {
		n := names[i]
		inputs = append(inputs,
			n,
			"Wide "+n,
			n+" of the Eagle",
			"Sanctified "+n+" of Grounding",
			strings.ToLower(n),
		)
	}

	for _, in := range inputs {
		wantBase, wantPrefix, wantSuffix := linearLongestItemBase(in)
		gotBase, gotPrefix, gotSuffix := indexItemBaseInName(in)
		testza.AssertEqual(t, wantBase, gotBase, "base for %q", in)
		testza.AssertEqual(t, wantPrefix, gotPrefix, "prefix for %q", in)
		testza.AssertEqual(t, wantSuffix, gotSuffix, "suffix for %q", in)
	}
}
