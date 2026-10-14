package calculator

import (
	"encoding/json"
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
	"github.com/Vilsol/go-pob/pob"
)

type configPinOption struct {
	Var                     string `json:"var"`
	Type                    string `json:"type"`
	Section                 string `json:"section"`
	Label                   string `json:"label"`
	DefaultState            any    `json:"defaultState"`
	DefaultIndex            int    `json:"defaultIndex"`
	DefaultPlaceholderState any    `json:"defaultPlaceholderState"`
	List                    []struct {
		Val   any    `json:"val"`
		Label string `json:"label"`
	} `json:"list"`
	HasApply  bool `json:"hasApply"`
	ApplyLine int  `json:"applyLine"`

	includeTransfigured                                       bool
	implyCond, implyCondList, implyMinionCond, implyEnemyCond any

	Predicates map[string]any `json:"-"`
}

var configPinPredicateKeys = []string{
	"ifNode", "ifOption", "ifCond", "ifMinionCond", "ifEnemyCond", "ifCondTrue", "ifMult", "ifEnemyMult",
	"ifStat", "ifEnemyStat", "ifTagType", "ifFlag", "ifMod", "ifSkill", "ifSkillFlag", "ifSkillData",
}

func loadConfigPinOptions(path string) ([]configPinOption, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var opts []configPinOption
	if err := json.Unmarshal(raw, &opts); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	var generic []map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	for i := range opts {
		opts[i].Predicates = map[string]any{}
		for _, k := range configPinPredicateKeys {
			if v, ok := generic[i][k]; ok {
				opts[i].Predicates[k] = v
			}
		}
		opts[i].includeTransfigured, _ = generic[i]["includeTransfigured"].(bool)
		opts[i].implyCond = generic[i]["implyCond"]
		opts[i].implyCondList = generic[i]["implyCondList"]
		opts[i].implyMinionCond = generic[i]["implyMinionCond"]
		opts[i].implyEnemyCond = generic[i]["implyEnemyCond"]
	}
	if len(opts) < 100 {
		return nil, fmt.Errorf("REFUSING: %s holds %d options; the pin has hundreds", path, len(opts))
	}
	return opts, nil
}

func (o configPinOption) configPinDefault() string {
	switch o.Type {
	case "check":
		if b, ok := o.DefaultState.(bool); ok && b {
			return "true"
		}
		return "false"
	case "list":
		idx := o.DefaultIndex
		if idx == 0 {
			idx = 1
		}
		if idx <= len(o.List) {
			return configValueString(o.List[idx-1].Val)
		}
		return ""
	default:
		if o.DefaultState != nil {
			return configValueString(o.DefaultState)
		}
		return "0"
	}
}

func configValueString(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case string:
		return x
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

func configInputString(in pob.Input) string {
	switch {
	case in.String != nil:
		return *in.String
	case in.Boolean != nil:
		return strconv.FormatBool(*in.Boolean)
	case in.Number != nil:
		return strconv.FormatFloat(*in.Number, 'g', -1, 64)
	}
	return ""
}

type configHandlerClass string

const (
	configHandled configHandlerClass = "handled"
	configDead    configHandlerClass = "dead"
	configAbsent  configHandlerClass = "absent"
)

func classifyConfigHandler(name string, handlers map[string]bool, order map[string]bool) configHandlerClass {
	switch {
	case handlers[name] && order[name]:
		return configHandled
	case handlers[name]:
		return configDead
	}
	return configAbsent
}

func goStringLiteralCounts(root string, dirs []string) (map[string]map[string]int, error) {
	counts := map[string]map[string]int{}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(path) //nolint:gosec // walking the repo
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
			rel, _ := filepath.Rel(root, path)
			fset := token.NewFileSet()
			file := fset.AddFile(path, -1, len(src))
			var s scanner.Scanner
			s.Init(file, src, nil, 0)
			for {
				_, tok, lit := s.Scan()
				if tok == token.EOF {
					break
				}
				if tok != token.STRING {
					continue
				}
				v, err := strconv.Unquote(lit)
				if err != nil {
					continue
				}
				if counts[v] == nil {
					counts[v] = map[string]int{}
				}
				counts[v][rel]++
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", dir, err)
		}
	}
	return counts, nil
}

type configCensusRow struct {
	opt                          configPinOption
	class                        configHandlerClass
	refsElsewhere                []string
	carriersAny, carriersNonDflt int
}

func configCensusRows(opts []configPinOption, handlers, order map[string]bool, literals map[string]map[string]int,
	carriers map[string][2]int) []configCensusRow {
	rows := make([]configCensusRow, 0, len(opts))
	for _, o := range opts {
		r := configCensusRow{opt: o, class: classifyConfigHandler(o.Var, handlers, order)}
		for file := range literals[o.Var] {
			if file != filepath.Join("calculator", "configs.go") {
				r.refsElsewhere = append(r.refsElsewhere, file)
			}
		}
		sort.Strings(r.refsElsewhere)
		r.carriersAny, r.carriersNonDflt = carriers[o.Var][0], carriers[o.Var][1]
		rows = append(rows, r)
	}
	return rows
}

func TestConfigOptionCensusClassifies(t *testing.T) {
	handlers := map[string]bool{"live": true, "notInOrder": true}
	order := map[string]bool{"live": true, "orderOnly": true}
	for name, want := range map[string]configHandlerClass{
		"live": configHandled, "notInOrder": configDead, "orderOnly": configAbsent, "zzNoSuchOption": configAbsent,
	} {
		if got := classifyConfigHandler(name, handlers, order); got != want {
			t.Errorf("%s: %s, want %s", name, got, want)
		}
	}

	realHandlers, realOrder := configHandlerSets()
	if classifyConfigHandler("conditionFullLife", realHandlers, realOrder) != configHandled {
		t.Fatal("conditionFullLife is not handled; the census reads the wrong maps")
	}
	delete(realOrder, "conditionFullLife")
	if got := classifyConfigHandler("conditionFullLife", realHandlers, realOrder); got != configDead {
		t.Errorf("conditionFullLife without its configOrder entry: %s, want dead", got)
	}

	lits, err := goStringLiteralCounts("..", []string{"calculator"})
	if err != nil {
		t.Fatal(err)
	}
	if lits["conditionFullLife"][filepath.Join("calculator", "configs.go")] == 0 {
		t.Fatal("the literal scan did not see conditionFullLife in configs.go")
	}
}

func configHandlerSets() (map[string]bool, map[string]bool) {
	handlers := map[string]bool{}
	for name := range configurations {
		handlers[name] = true
	}
	for name := range configurationsWithBuild {
		handlers[name] = true
	}
	order := map[string]bool{}
	for _, name := range configOrder {
		order[name] = true
	}
	return handlers, order
}

// TestConfigOptionCensus is `task config-census`: one row per pin option.
func TestConfigOptionCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census; `task config-census`")
	in := requireTaskInput(t, "config-census", "CONFIG_OPTIONS", "CONFIG_CENSUS_OUT")
	optsPath, out := in[0], in[1]
	opts, err := loadConfigPinOptions(optsPath)
	if err != nil {
		t.Fatal(err)
	}
	handlers, order := configHandlerSets()
	lits, err := goStringLiteralCounts("..", []string{"calculator", "pob", "builds", "wasm", "moddb", "mod"})
	if err != nil {
		t.Fatal(err)
	}

	defaults := map[string]string{}
	for _, o := range opts {
		defaults[o.Var] = o.configPinDefault()
	}
	corpusDir, names := corpusNamesForLoadedData(t)
	perBuild := make([]map[string]bool, len(names))
	runConcurrently(len(names), evalWorkers(), func(i int, _ *EnvironmentCache) {
		raw, err := os.ReadFile(filepath.Join(corpusDir, names[i]))
		if err != nil {
			return
		}
		build, err := parsecache.ParseCached(raw)
		if err != nil {
			return
		}
		seen := map[string]bool{}
		for _, in := range build.Config.Inputs {
			v := configInputString(in)
			seen[in.Name] = seen[in.Name] || v != defaults[in.Name]
		}
		perBuild[i] = seen
	})
	carriers := map[string][2]int{}
	parsed := 0
	for _, seen := range perBuild {
		if seen == nil {
			continue
		}
		parsed++
		for name, nonDefault := range seen {
			c := carriers[name]
			c[0]++
			if nonDefault {
				c[1]++
			}
			carriers[name] = c
		}
	}
	if len(carriers) == 0 {
		t.Fatalf("REFUSING: %d corpus builds parsed and none carries an <Input>", parsed)
	}

	rows := configCensusRows(opts, handlers, order, lits, carriers)
	var b strings.Builder
	b.WriteString("var\ttype\tsection\tpin_apply\tapply_line\thandler\trefs_elsewhere\tcarriers_any\tcarriers_nondefault\tpredicates\n")
	tally := map[string]int{}
	for _, r := range rows {
		preds := make([]string, 0, len(r.opt.Predicates))
		for _, k := range configPinPredicateKeys {
			if v, ok := r.opt.Predicates[k]; ok {
				j, _ := json.Marshal(v)
				preds = append(preds, k+"="+string(j))
			}
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%t\t%d\t%s\t%s\t%d\t%d\t%s\n", r.opt.Var, r.opt.Type, r.opt.Section,
			r.opt.HasApply, r.opt.ApplyLine, r.class, strings.Join(r.refsElsewhere, ","), r.carriersAny,
			r.carriersNonDflt, strings.Join(preds, " "))
		key := string(r.class)
		if len(r.refsElsewhere) > 0 {
			key += "+refs"
		}
		if !r.opt.HasApply {
			key += " (pin: no apply)"
		}
		tally[key]++
		if r.carriersAny > 0 {
			tally["corpus-carried (any value)"]++
		}
		if r.carriersNonDflt > 0 {
			tally["corpus-carried (non-default)"]++
		}
	}

	pin := map[string]bool{}
	for _, o := range opts {
		pin[o.Var] = true
	}
	var goOnly []string
	for name := range handlers {
		if !pin[name] {
			goOnly = append(goOnly, name)
		}
	}
	sort.Strings(goOnly)

	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "census.tsv"), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var s strings.Builder
	fmt.Fprintf(&s, "%d pin options; %d corpus builds parsed\n", len(opts), parsed)
	for _, k := range keys {
		fmt.Fprintf(&s, "  %-40s %4d\n", k, tally[k])
	}
	fmt.Fprintf(&s, "handler keys not in the pin: %d  %s\n", len(goOnly), strings.Join(goOnly, " "))
	if err := os.WriteFile(filepath.Join(out, "census-summary.txt"), []byte(s.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	fmt.Print(s.String())
	fmt.Printf("census written to %s (census.tsv, census-summary.txt)\n", out)
}
