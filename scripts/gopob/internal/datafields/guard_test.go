//go:build !js

package datafields

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/fields"
	"golang.org/x/mod/semver"
)

const repoRoot = "../../../.."

func committedList(game Game, version string) (*List, error) {
	return Published(game, version)
}

// TestReadFieldsAreInTheCommittedList fails, naming asset and key path, when go-pob reads a go-pob-data field the published list would drop from the .min assets.
func TestReadFieldsAreInTheCommittedList(t *testing.T) {
	if os.Getenv("OPERATOR") != "true" {
		t.Skip("set OPERATOR=true (mise run data-fields-guard, run by data-regen and CI's lint job) — a whole-module analysis")
	}
	for _, game := range Games {
		t.Run(game.Name, func(t *testing.T) { guardGame(t, game) })
	}
}

func guardGame(t *testing.T, game Game) {
	res, err := Analyze(repoRoot, game)
	if err != nil {
		t.Fatal(err)
	}
	fresh := Build(res, DataModule+"@"+res.DataModuleVersion, res.DefaultVersion)
	committed, err := committedList(game, res.DefaultVersion)
	if err != nil {
		t.Fatalf("%v; run `mise run data-fields -- --game %s --list FILE` and send it upstream", err, game.Name)
	}
	if committed.Format != Format && committed.Format != FormatV1 || fields.Format != Format {
		t.Errorf("list format %q, go-pob-data reads %q, this tool writes %q", committed.Format, fields.Format, Format)
	}
	listed, ok := strings.CutPrefix(committed.DataModule, DataModule+"@")
	if !ok || !semver.IsValid(listed) || semver.Compare(listed, res.DataModuleVersion) > 0 {
		t.Errorf("the list was made for %s, go.mod requires %s", committed.DataModule, fresh.DataModule)
	}
	for _, m := range Missing(committed, fresh) {
		t.Error(m)
	}
	if len(fresh.Assets) == 0 {
		t.Fatal("the analysis found no loaded assets, so an empty comparison would pass")
	}
}
