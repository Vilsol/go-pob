package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const (
	oeGemsPy    = "oracle/gems_emit.py"
	gemsParser  = "parser.go"
	gemsOutGo   = "out.go"
	gemsRenames = `package builds

// pobGemIDtoGameGemIDs is the single definition of the divergence.
var pobGemIDtoGameGemIDs = map[string]string{
	"Metadata/Items/Gems/Smite": "Metadata/Items/Gems/SkillGemSmite",
	"Metadata/Items/Gems/Gone":  "Metadata/Items/Gems/AlsoGone",
}
`
)

func gemsDump() string {
	return `{
  "pobVersion": "2.67.2-fixture",
  "tagNames": ["spell", "area", "aura", "attack", "melee", "projectile", "duration",
               "trap", "mine", "totem", "brand", "support"],
  "gems": {
    "z1": {"id": "Metadata/Items/Gems/Smite", "name": "Smite", "loc": 40,
           "tags": ["spell", "aura"], "grantedEffectId": "Smite"},
    "a2": {"id": "Metadata/Items/Gems/SkillGemFireball", "name": "Fireball", "loc": 10,
           "tags": ["spell", "projectile"], "grantedEffectId": "Fireball"},
    "m3": {"id": "Metadata/Items/Gems/SkillGemVaalFireball", "name": "Vaal Fireball", "loc": 20,
           "tags": ["spell", "projectile"], "grantedEffectId": "VaalFireball",
           "secondaryGrantedEffectId": "Fireball"},
    "zb4": {"id": "Metadata/Items/Gems/SupportTrap", "name": "Trap", "loc": 30,
           "tags": ["support", "trap"], "gameId": "Metadata/Items/Gems/SupportTrapGame",
           "grantedEffectId": "SupportTrap"},
    "ac5": {"id": "Metadata/Items/Gems/SupportTrapAltX", "name": "Trap of Skewering", "loc": 35,
           "tags": ["support", "trap"], "gameId": "Metadata/Items/Gems/SupportTrapGame",
           "grantedEffectId": "SupportTrapAltX"},
    "d6": {"id": "Metadata/Items/Gems/ReqGem", "name": "Req Gem", "loc": 50,
           "tags": ["attack"], "reqStr": 68, "reqDex": 0, "reqInt": 12, "naturalMaxLevel": 5},
    "g10": {"id": "Metadata/Items/Gems/Filler10", "name": "Filler 10", "loc": 70, "tags": ["melee"],
           "grantedEffectId": "SharedFx"},
    "g09": {"id": "Metadata/Items/Gems/Filler09", "name": "Filler 09", "loc": 70, "tags": ["melee"],
           "secondaryGrantedEffectId": "SharedFx"},
    "g08": {"id": "Metadata/Items/Gems/Filler08", "name": "Filler 08", "loc": 70, "tags": ["melee"]},
    "g07": {"id": "Metadata/Items/Gems/Filler07", "name": "Filler 07", "loc": 70, "tags": ["melee"]},
    "g06": {"id": "Metadata/Items/Gems/Filler06", "name": "Filler 06", "loc": 70, "tags": ["melee"]},
    "g05": {"id": "Metadata/Items/Gems/Filler05", "name": "Filler 05", "loc": 70, "tags": ["melee"]},
    "g04": {"id": "Metadata/Items/Gems/Filler04", "name": "Filler 04", "loc": 70, "tags": ["melee"]},
    "g03": {"id": "Metadata/Items/Gems/Filler03", "name": "Filler 03", "loc": 70, "tags": ["melee"]},
    "g02": {"id": "Metadata/Items/Gems/Filler02", "name": "Filler 02", "loc": 70, "tags": ["melee"]},
    "g01": {"id": "Metadata/Items/Gems/Filler01", "name": "Filler 01", "loc": 70, "tags": ["melee"]}
  }
}`
}

func gemsSandbox(t *testing.T, repo, dump string) func() (string, error) {
	t.Helper()
	src := map[string]string{}
	for _, rel := range []string{oeCommonPy, oeGemsPy} {
		b := difftest.ReadOriginal(t, repo, rel)
		if b == nil {
			continue
		}
		src[rel] = string(b)
	}
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{"dump.json": dump, gemsParser: gemsRenames}
		for rel, body := range src {
			files[rel] = body
		}
		for rel, body := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func gemsPut(sb, rel, body string) error {
	if err := os.WriteFile(filepath.Join(sb, rel), []byte(body), 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func gemsOneRow(row string) string {
	return `{"pobVersion": "v", "tagNames": ["a"], "gems": {"k1": ` + row + `}}`
}

func TestOracleGemsEmit(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := gemsSandbox(t, repo, gemsDump())
	s := oeSpec(bin, "gems-emit", "dump.json", gemsOutGo, gemsParser)
	oeRunFixtures(t, s, []difftest.Fixture{
		{
			Name:    "no-gems",
			Refusal: "the dump has no gems. Re-run `go run ./scripts/gamedatadump gems`.",
			Break:   func(sb string) error { return gemsPut(sb, "dump.json", `{"gems": {}}`) },
		},
		{
			Name:    "gems-is-an-array",
			Refusal: "the dump's 'gems' is not a JSON object",
			Break:   func(sb string) error { return gemsPut(sb, "dump.json", `{"gems": [{"id": "A"}]}`) },
		},
		{
			Name:    "a-gem-is-not-an-object",
			Refusal: "gem k1 is not a JSON object",
			Break:   func(sb string) error { return gemsPut(sb, "dump.json", gemsOneRow(`"scalar"`)) },
		},
		{
			Name:    "a-gem-has-no-id",
			Refusal: "gem k1 has no 'id' string",
			Break:   func(sb string) error { return gemsPut(sb, "dump.json", gemsOneRow(`{"name": "N"}`)) },
		},
		{
			Name:    "a-gem-has-no-name",
			Refusal: "A: has no 'name' string",
			Break:   func(sb string) error { return gemsPut(sb, "dump.json", gemsOneRow(`{"id": "A"}`)) },
		},
		{
			Name:    "tags-is-a-string",
			Refusal: "A: 'tags' is not a list of strings",
			Break: func(sb string) error {
				return gemsPut(sb, "dump.json", gemsOneRow(`{"id": "A", "name": "N", "tags": "spell"}`))
			},
		},
		{
			Name:    "a-tag-is-not-a-string",
			Refusal: "A: 'tags' is not a list of strings",
			Break: func(sb string) error {
				return gemsPut(sb, "dump.json", gemsOneRow(`{"id": "A", "name": "N", "tags": ["ok", 7]}`))
			},
		},
		{
			Name:    "a-gem-has-no-loc",
			Refusal: "A: the dumper found no declaring line to cite",
			Break: func(sb string) error {
				return gemsPut(sb, "dump.json", gemsOneRow(`{"id": "A", "name": "N", "tags": ["x"]}`))
			},
		},
		{
			Name:    "a-loc-is-not-a-number",
			Refusal: "A: declaring line 'nine' is not a number",
			Break: func(sb string) error {
				return gemsPut(sb, "dump.json",
					gemsOneRow(`{"id": "A", "name": "N", "tags": ["x"], "loc": "nine"}`))
			},
		},
		{
			Name:   "the-rename-source-is-crlf",
			Passes: true,
			Break: func(sb string) error {
				return gemsPut(sb, gemsParser, strings.ReplaceAll(gemsRenames, "\n", "\r\n"))
			},
		},
		{
			Name:    "a-rename-row-contains-a-vertical-tab",
			Refusal: "parser.go: cannot parse pobGemIDtoGameGemIDs row",
			Break: func(sb string) error {
				return gemsPut(sb, gemsParser, "package builds\n\n"+
					"var pobGemIDtoGameGemIDs = map[string]string{\n\t\"a\": \"b\",\vjunk\n}\n")
			},
		},
		{
			Name:    "the-rename-source-is-missing",
			Refusal: "cannot read the rename table's source parser.go: No such file or directory",
			Break:   func(sb string) error { return os.Remove(filepath.Join(sb, gemsParser)) },
		},
		{
			Name:    "the-rename-source-declares-no-table",
			Refusal: "parser.go declares no `pobGemIDtoGameGemIDs = map[string]string{`",
			Break:   func(sb string) error { return gemsPut(sb, gemsParser, "package builds\n") },
		},
		{
			Name:    "the-rename-table-has-no-closing-brace",
			Refusal: "parser.go: pobGemIDtoGameGemIDs has no closing brace",
			Break: func(sb string) error {
				return gemsPut(sb, gemsParser,
					"package builds\n\nvar pobGemIDtoGameGemIDs = map[string]string{\n\t\"a\": \"b\",\n")
			},
		},
		{
			Name:    "a-rename-row-is-unparseable",
			Refusal: `parser.go: cannot parse pobGemIDtoGameGemIDs row 'nonsense'`,
			Break: func(sb string) error {
				return gemsPut(sb, gemsParser, "package builds\n\n"+
					"var pobGemIDtoGameGemIDs = map[string]string{\n\tnonsense\n}\n")
			},
		},
		{
			Name:    "the-rename-table-is-empty",
			Refusal: "parser.go: pobGemIDtoGameGemIDs is empty",
			Break: func(sb string) error {
				return gemsPut(sb, gemsParser, "package builds\n\n"+
					"var pobGemIDtoGameGemIDs = map[string]string{\n}\n")
			},
		},
		{
			Name: "a-rename-collides-with-a-declared-id",
			Refusal: "Metadata/Items/Gems/Smite rewrites to Metadata/Items/Gems/SkillGemSmite, " +
				"which Metadata/Items/Gems/SkillGemSmite already occupies.",
			Break: func(sb string) error {
				return gemsPut(sb, "dump.json", `{"pobVersion": "v", "tagNames": ["a"], "gems": {
					"k1": {"id": "Metadata/Items/Gems/SkillGemSmite", "name": "S", "loc": 1, "tags": ["a"]},
					"k2": {"id": "Metadata/Items/Gems/Smite", "name": "S2", "loc": 2, "tags": ["a"]}}}`)
			},
		},
		{
			Name:    "a-shadowed-row-has-no-granted-effect",
			Refusal: "B shares the gameId G with A and declares no grantedEffectId",
			Break: func(sb string) error {
				return gemsPut(sb, "dump.json", `{"pobVersion": "v", "tagNames": ["a"], "gems": {
					"k1": {"id": "A", "name": "A", "loc": 1, "tags": ["a"], "gameId": "G",
					       "grantedEffectId": "AFx"},
					"k2": {"id": "B", "name": "B", "loc": 2, "tags": ["a"], "gameId": "G"}}}`)
			},
		},
		{
			Name:    "two-gems-claim-one-granted-effect-as-PRIMARY",
			Refusal: "granted effect Shared is claimed as the PRIMARY of both",
			Break: func(sb string) error {
				return gemsPut(sb, "dump.json", `{"pobVersion": "v", "tagNames": ["a"], "gems": {
					"k1": {"id": "A", "name": "A", "loc": 1, "tags": ["a"], "grantedEffectId": "Shared"},
					"k2": {"id": "B", "name": "B", "loc": 2, "tags": ["a"], "grantedEffectId": "Shared"}}}`)
			},
		},
	}, mk)
}

func TestOracleGemsEmitUsage(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := gemsSandbox(t, repo, gemsDump())
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no-arguments", nil},
		{"two-arguments", []string{"dump.json", gemsOutGo}},
		{"four-arguments", []string{"dump.json", gemsOutGo, gemsParser, "extra"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := oeSpec(bin, "gems-emit", tc.args...)
			s.Files = nil
			difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

// TestOracleGemsEmitOrderIsStableAcrossRuns: THE ORDER CONTROL, REPEATED.
func TestOracleGemsEmitOrderIsStableAcrossRuns(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := gemsSandbox(t, repo, gemsDump())
	s := oeSpec(bin, "gems-emit", "dump.json", gemsOutGo, gemsParser)
	s.RefusalSites = 1
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), s, difftest.Fixture{Name: "happy"}, mk)
		})
	}
}
