package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const cfgPy = "scripts/config_tools.py"

func cfgLua() string {
	return strings.Join([]string{
		`-- ConfigOptions.lua fixture`,
		`return {`,
		`	{ section = "Alpha" },`,
		`	{ var = "ctPlain", type = "check", section = "Alpha",`,
		`	  apply = function(val, modList) modList:NewMod("A", "FLAG", val) end },`,
		`	{ var = "ctTooltip", type = "count", section = "Alpha",`,
		"	  tooltip = [[Bosses' damage is increased { not a brace } here",
		`	  and it continues onto a second line { still not a brace }`,
		`	  and a third]],`,
		`	  apply = function(val, modList) modList:NewMod("B", "BASE", val) end },`,
		`	{ var = "ctLevelBracket", type = "list", section = "Beta",`,
		"	  tooltip = [=[a level-1 bracket with ]] inside it { unbalanced]=],",
		`	  apply = function(val, modList) modList:NewMod("C", "BASE", val) end },`,
		`	--[[ a long COMMENT holding { braces } and an apostrophe ' that must not open a string ]]`,
		`	{ var = "ctLongComment", type = "check", section = "Beta",`,
		`	  defaultState = true,`,
		`	  apply = function(val, modList) modList:NewMod("D", "FLAG", val) end },`,
		`	{ var = "ctStrayQuote", type = "check", section = "Beta",`,
		`	  hint = 'an unclosed apostrophe`,
		`	  apply = function(val, modList) modList:NewMod("K", "FLAG", val) end },`,
		`	{ var = "ctBracesInString", type = "count", section = "Beta",`,
		`	  label = "a string with { and } in it",`,
		`	  defaultPlaceholderState = true,`,
		`	  apply = function(val, modList) modList:NewMod("E", "BASE", val) end },`,
		`	{ var = "ctEscaped", type = "check", section = "Gamma",`,
		`	  label = "an escaped \" quote and a { brace",`,
		`	  apply = function(val, modList) modList:NewMod("F", "FLAG", val) end },`,
		`	-- a line comment with a { brace and then an unbalanced apostrophe '`,
		`	{ var = "ctNoApply", type = "list", section = "Gamma" },`,
		`	{ var = "ctNoType", section = "Gamma",`,
		`	  apply = function(val, modList) modList:NewMod("G", "BASE", val) end },`,
		`	{ var = "ctDeadOne", type = "check", section = "Gamma",`,
		`	  apply = function(val, modList) modList:NewMod("H", "FLAG", val) end },`,
		`	{ var = "ctDeadTwo", type = "count", section = "Gamma",`,
		`	  apply = function(val, modList) modList:NewMod("I", "BASE", val) end },`,
		`	{ var = "ctBothLiveAndDead", type = "check", section = "Gamma",`,
		`	  apply = function(val, modList) modList:NewMod("J", "FLAG", val) end },`,
		`	{ var = "ctAbsentOne", type = "check", section = "Delta" },`,
		`	{ var = "ctAbsentTwo", type = "count", section = "Delta" },`,
		`	{ var = "ctAbsentThree", type = "list", section = "Delta" },`,
		`	{ var = "ctAbsentFour", type = "check", section = "Delta" },`,
		`	{ var = "ctAbsentFive", type = "count", section = "Delta" },`,
		`	{ var = "ctAbsentSix", type = "list", section = "Delta" },`,
		`	{ var = "ctAbsentSeven", type = "check", section = "Delta" },`,
		`	{ var = "ctAbsentEight", type = "count", section = "Delta" },`,
		`	{ var = "ctAbsentNine", type = "list", section = "Delta" },`,
		`	{ var = "ctAbsentTen", type = "check", section = "Delta" },`,
		`	{ var = "ctAbsentEleven", type = "count", section = "Delta" },`,
		`	{ var = "ctAbsentTwelve", type = "list", section = "Delta" },`,
		`	{ var = "ctAbsentThirteen", type = "check", section = "Delta" },`,
		`	{ var = "ctAbsentFourteen", type = "count", section = "Delta" },`,
		`}`,
		``,
	}, "\n")
}

func cfgConfigsGo() string {
	return strings.Join([]string{
		`package calculator`,
		``,
		`var configurations = map[string]ConfigApplyFunc{`,
		"\t\"ctPlain\": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {",
		"\t\tmodList.NewFlag(\"A\", val)",
		"\t},",
		"\t\"ctTooltip\": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {",
		"\t\tmodList.NewBase(\"B\", val)",
		"\t},",
		"\t\"ctBothLiveAndDead\": func(val interface{}, modList *moddb.ModList, enemyModList *moddb.ModList) {",
		"\t\tmodList.NewFlag(\"J\", val)",
		"\t},",
		`	/* a prefix before the dead handler, and no suffix after it`,
		"\t\t\"ctDeadOne\": func(val interface{}, modList *ModList, enemyModList *ModList) {",
		"\t\t\tmodList.NewFlag(\"H\", val)",
		"\t\t},",
		`	*/`,
		`	/*`,
		"\t   \"ctDeadTwo\": func(val interface{}, modList *ModList, enemyModList *ModList) {",
		"\t      modList.NewBase(\"I\", val)",
		"\t   },",
		`	a suffix after the dead handler and no prefix before it`,
		`	*/`,
		`	/* an older draft of the same dead handler, in its own block`,
		"\t\t\"ctDeadTwo\": func(val interface{}, modList *ModList, enemyModList *ModList) {",
		"\t\t\tmodList.NewBase(\"I-old\", val)",
		"\t\t},",
		`	*/`,
		`	/* a stale draft repeating a LIVE key, which must not count as dead`,
		"\t\t\"ctBothLiveAndDead\": func(val interface{}, modList *ModList, enemyModList *ModList) {",
		"\t\t\tmodList.NewFlag(\"J\", val)",
		"\t\t},",
		`	*/`,
		`}`,
		``,
	}, "\n")
}

func cfgCorpusBuild(vars ...string) string {
	var b strings.Builder
	b.WriteString("<PathOfBuilding><Config>\n")
	for _, v := range vars {
		fmt.Fprintf(&b, "\t<Input name=\"%s\" number=\"1\"/>\n", v)
	}
	b.WriteString("</Config></PathOfBuilding>\n")
	return b.String()
}

func cfgSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, cfgPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			cfgPy:                                    string(py),
			"pob/src/Modules/ConfigOptions.lua":      cfgLua(),
			"calculator/configs.go":                  cfgConfigsGo(),
			"emptypob/src/Modules/ConfigOptions.lua": "local notTheTable = 1\n",
			"corpus/a.xml": cfgCorpusBuild("ctDeadOne", "ctDeadTwo", "ctAbsentOne", "ctAbsentTwo",
				"ctUndeclaredZ", "ctUndeclaredA"),
			"corpus/b.xml":      cfgCorpusBuild("ctDeadOne", "ctDeadTwo", "ctAbsentOne", "ctAbsentTwo"),
			"corpus/c.xml":      cfgCorpusBuild("ctAbsentThree", "ctUndeclaredA"),
			"corpus/notxml.txt": "ignored\n",
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
		cmd := exec.CommandContext(t.Context(), "git", "init", "-q")
		cmd.Dir = sb
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("sandbox: git init: %w: %s", err, out)
		}
		return sb, nil
	}
}

func cfgBase(extra ...string) []string {
	return append([]string{"--pob", "pob", "--repo", ".", "--corpus", "corpus"}, extra...)
}

func TestConfigToolsPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cfgSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, cfgPy)

	base := cfgBase("inventory")
	spec := difftest.Spec{
		Tool:         "config-tools",
		PythonScript: cfgPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"config-tools"}, base...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 2,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-command", Refusal: "REFUSING: a command is required",
				Break: func(string) error { return nil }},
			{Name: "unknown-command", Refusal: "REFUSING: unknown command 'bogus'; expected one of extract, inventory, compare, splice",
				Break: func(string) error { return nil }},
			{Name: "flag-without-value", Refusal: "REFUSING: --var needs a value",
				Break: func(string) error { return nil }},
			{Name: "unknown-flag", Refusal: "REFUSING: unknown argument '--bogus'",
				Break: func(string) error { return nil }},
			{Name: "second-positional", Refusal: "REFUSING: unexpected argument 'extra'",
				Break: func(string) error { return nil }},
			{Name: "config-options-missing", Refusal: "ConfigOptions.lua is not readable: No such file or directory",
				Break: func(sb string) error {
					return os.Remove(filepath.Join(sb, "pob/src/Modules/ConfigOptions.lua"))
				}},
			{Name: "no-return-table", Refusal: "has no `return {` at column 0",
				Break: func(string) error { return nil }},
			{Name: "configs-go-missing", Refusal: "configs.go is not readable: No such file or directory",
				Break: func(sb string) error {
					return os.Remove(filepath.Join(sb, "calculator/configs.go"))
				}},
			{Name: "handler-table-renamed", Refusal: "has no `var configurations = map[string]ConfigApplyFunc{`",
				Break: func(sb string) error {
					s := strings.Replace(cfgConfigsGo(), "var configurations = map[string]ConfigApplyFunc{",
						"var somethingElse = map[string]ConfigApplyFunc{", 1)
					return os.WriteFile(filepath.Join(sb, "calculator/configs.go"), []byte(s), 0o600)
				}},
			{Name: "config-options-not-utf8", Refusal: "ConfigOptions.lua is not valid UTF-8",
				Break: func(sb string) error {
					bad := strings.Replace(cfgLua(), `section = "Alpha" }`, "section = \"Al\xffpha\" }", 1)
					return os.WriteFile(filepath.Join(sb, "pob/src/Modules/ConfigOptions.lua"),
						[]byte(bad), 0o600)
				}},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	argsFor := map[string][]string{
		"no-command":         {"--pob", "pob"},
		"unknown-command":    cfgBase("bogus"),
		"flag-without-value": {"--pob", "pob", "extract", "--var"},
		"unknown-flag":       cfgBase("extract", "--bogus"),
		"second-positional":  cfgBase("extract", "extra"),
		"no-return-table":    {"--pob", "emptypob", "--repo", ".", "--corpus", "corpus", "extract"},
	}

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			if a, ok := argsFor[f.Name]; ok {
				s.PythonArgs = a
			}
			s.GoArgs = append([]string{"config-tools"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

func TestConfigToolsModes(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := cfgSandbox(t, repo)
	for _, tc := range []struct {
		name  string
		args  []string
		files []string
	}{
		{"inventory", cfgBase("inventory"), nil},
		{"inventory-no-corpus", []string{"--pob", "pob", "--repo", ".", "--corpus", "nope", "inventory"}, nil},
		{"extract-all", cfgBase("extract"), nil},
		{"extract-one", cfgBase("extract", "--var", "ctTooltip"), nil},
		{"extract-no-apply", cfgBase("extract", "--var", "ctNoApply"), nil},
		{"extract-no-type", cfgBase("extract", "--var", "ctNoType"), nil},
		{"extract-missing-var", cfgBase("extract", "--var", "ctNoSuchVar"), nil},
		{"extract-several", cfgBase("extract", "--var", "ctPlain", "--var", "ctEscaped"), nil},
		{"compare-live", cfgBase("compare", "--var", "ctPlain"), nil},
		{"compare-dead", cfgBase("compare", "--var", "ctDeadOne"), nil},
		{"compare-both", cfgBase("compare", "--var", "ctBothLiveAndDead"), nil},
		{"compare-absent", cfgBase("compare", "--var", "ctAbsentOne"), nil},
		{"splice-dry-prefix-only", cfgBase("splice", "--var", "ctDeadOne", "--dry-run"), nil},
		{"splice-dry-suffix-only", cfgBase("splice", "--var", "ctDeadTwo", "--dry-run"), nil},
		{"splice-not-found", cfgBase("splice", "--var", "ctNoSuchVar", "--dry-run"), nil},
		{"splice-writes", cfgBase("splice", "--var", "ctDeadOne"), []string{"calculator/configs.go"}},
		{"splice-writes-suffix", cfgBase("splice", "--var", "ctDeadTwo"), []string{"calculator/configs.go"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "config-tools",
				PythonScript: cfgPy,
				PythonArgs:   tc.args,
				GoBin:        bin,
				GoArgs:       append([]string{"config-tools"}, tc.args...),
				Stderr:       difftest.StderrExact,
				Files:        tc.files,
			}, difftest.Fixture{Name: tc.name}, mk)
		})
	}
}

func TestConfigToolsPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := cfgSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "config-tools",
		PythonScript: cfgPy,
		PythonArgs:   cfgBase("inventory"),
	}, mk, []string{"0", "12345", "999983"})
}
