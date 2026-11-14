package mcp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/pob"
)

// config_options.json is ConfigOptions.lua's option list, written by oracle/config_options.lua from the pinned PoB
// (mise run mcp-data).
//
//go:embed config_options.json
var configOptionsJSON []byte

type configListEntry struct {
	Val   any    `json:"val"`
	Label string `json:"label"`
}

type configOption struct {
	Var          string            `json:"var"`
	Type         string            `json:"type"`
	Label        string            `json:"label"`
	Section      string            `json:"section"`
	List         []configListEntry `json:"list"`
	DefaultIndex int               `json:"defaultIndex"`
	DefaultState any               `json:"defaultState"`
}

func (o configOption) cleanLabel() string {
	return strings.TrimSuffix(strings.TrimSpace(stripColor(o.Label)), ":")
}

func stripColor(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '^' && i+1 < len(s) {
			if s[i+1] == 'x' && i+7 < len(s) {
				i += 7
				continue
			}
			if s[i+1] >= '0' && s[i+1] <= '9' {
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

var configOptions = sync.OnceValue(func() []configOption {
	var all, out []configOption
	_ = json.Unmarshal(configOptionsJSON, &all)
	seen := map[string]bool{}
	for _, o := range all {
		if o.Var == "" || o.Type == "" || seen[o.Var] {
			continue
		}
		seen[o.Var] = true
		out = append(out, o)
	}
	return out
})

func (o configOption) defaultValue() any {
	if o.Type == "list" && len(o.List) > 0 {
		i := max(o.DefaultIndex, 1) - 1
		if i < len(o.List) {
			return o.List[i].Val
		}
	}
	if o.DefaultState != nil {
		return o.DefaultState
	}
	if o.Type == "check" {
		return false
	}
	return nil
}

func resolveConfig(name string) (configOption, *ToolError) {
	opts := configOptions()
	for _, o := range opts {
		if o.Var == name {
			return o, nil
		}
	}
	o, ok, hits := resolveName(name, opts, configOption.cleanLabel, func(o configOption) string { return o.Var })
	if ok {
		return o, nil
	}
	if len(hits) > 0 {
		e := errf(CodeAmbiguous, "%q matches several config options.", name)
		for _, h := range hits[:min(len(hits), 8)] {
			e.Candidates = append(e.Candidates, map[string]any{"var": h.Var, "label": h.cleanLabel()})
		}
		return configOption{}, e
	}
	var names []string
	for _, o := range opts {
		names = append(names, o.Var)
	}
	return configOption{}, notFound("config option", name, suggest(name, names, 5), "Retry with a var from search_data kind=config.")
}

func listValue(o configOption, value any) (configListEntry, *ToolError) {
	s := fmt.Sprint(value)
	for _, l := range o.List {
		if fmt.Sprint(l.Val) == s || norm(stripColor(l.Label)) == norm(s) {
			return l, nil
		}
	}
	e := errf(CodeInvalidValue, "%q is not a value of %s.", s, o.Var)
	for _, l := range o.List {
		e.Allowed = append(e.Allowed, map[string]any{"value": l.Val, "label": stripColor(l.Label)})
	}
	return configListEntry{}, e
}

// configInput converts a value to the option's stored Input; isDefault reports a list value equal to its default,
// which PoB does not store.
func configInput(o configOption, value any) (pob.Input, bool, *ToolError) {
	in := pob.Input{Name: o.Var}
	switch o.Type {
	case "check":
		v, ok := value.(bool)
		if !ok {
			if s, isStr := value.(string); isStr {
				v, ok = s == "true", s == "true" || s == "false"
			}
		}
		if !ok {
			e := errf(CodeInvalidValue, "%s takes true or false.", o.Var)
			e.Allowed = []any{true, false}
			return in, false, e
		}
		in.Boolean = &v
	case "list":
		l, e := listValue(o, value)
		if e != nil {
			return in, false, e
		}
		switch v := l.Val.(type) {
		case float64:
			in.Number = &v
		default:
			s := fmt.Sprint(v)
			in.String = &s
		}
		return in, fmt.Sprint(l.Val) == fmt.Sprint(o.defaultValue()), nil
	case "count", "integer", "float", "countAllowZero":
		var f float64
		switch v := value.(type) {
		case float64:
			f = v
		case string:
			p, err := strconv.ParseFloat(v, 64)
			if err != nil {
				e := errf(CodeInvalidValue, "%s takes a number.", o.Var)
				e.Allowed = []any{"a number"}
				return in, false, e
			}
			f = p
		default:
			e := errf(CodeInvalidValue, "%s takes a number.", o.Var)
			e.Allowed = []any{"a number"}
			return in, false, e
		}
		in.Number = &f
	default:
		s := fmt.Sprint(value)
		in.String = &s
	}
	return in, false, nil
}

type buildOp struct {
	Op     string `json:"op"`
	Level  *int   `json:"level"`
	Bandit string `json:"bandit"`
	Major  string `json:"major"`
	Minor  string `json:"minor"`
	Var    string `json:"var"`
	Value  any    `json:"value"`
	Text   string `json:"text"`
}

type editBuildIn struct {
	Handle string    `json:"handle"`
	DryRun bool      `json:"dry_run"`
	Watch  []string  `json:"watch"`
	Ops    []buildOp `json:"ops"`
}

var buildOps = []string{"set_level", "set_bandit", "set_pantheon", "set_config", "unset_config", "set_notes", "append_notes"}

func (s *Server) editBuild(args json.RawMessage) (any, *ToolError) {
	var in editBuildIn
	if e := decode(args, schemaEditBuild, &in); e != nil {
		return nil, e
	}
	if e := checkOps(len(in.Ops), 20); e != nil {
		return nil, e
	}
	return s.runEdit(in.Handle, in.DryRun, in.Watch, len(in.Ops), func(ec *editCtx, i int) *ToolError {
		return applyBuildOp(ec, in.Ops[i])
	})
}

func pantheonValue(varName, value string) (string, *ToolError) {
	o, e := resolveConfig(varName)
	if e != nil {
		return "", e
	}
	l, e := listValue(o, value)
	if e != nil {
		// "Soul of the Brine King" and "Brine King" both name TheBrineKing
		for _, x := range o.List {
			if strings.Contains(norm(stripColor(x.Label)), norm(value)) {
				return fmt.Sprint(x.Val), nil
			}
		}
		return "", e
	}
	return fmt.Sprint(l.Val), nil
}

func applyBuildOp(ec *editCtx, op buildOp) *ToolError {
	b := ec.b
	switch op.Op {
	case "set_level":
		if op.Level == nil || *op.Level < 1 || *op.Level > 100 {
			e := errf(CodeInvalidValue, "level must be 1-100.")
			e.Allowed = []any{"1-100"}
			return e
		}
		b.SetLevel(*op.Level)
		b.SetCharacterLevelAutoMode(false)
		ec.note(fmt.Sprintf("set_level %d", *op.Level))
	case "set_bandit":
		for _, v := range []string{"None", "Alira", "Kraityn", "Oak"} {
			if norm(v) == norm(op.Bandit) || (norm(op.Bandit) == "kill all" && v == "None") {
				b.Build.Bandit = v
				b.SetConfigOption(stringInput("bandit", v))
				ec.note("set_bandit " + v)
				return nil
			}
		}
		e := errf(CodeInvalidValue, "No bandit %q.", op.Bandit)
		e.Allowed = []any{"None", "Alira", "Kraityn", "Oak"}
		return e
	case "set_pantheon":
		if op.Major == "" && op.Minor == "" {
			return invalidArg("set_pantheon needs major and/or minor.", nil)
		}
		if op.Major != "" {
			v, e := pantheonValue("pantheonMajorGod", op.Major)
			if e != nil {
				return e
			}
			b.Build.PantheonMajorGod = v
			b.SetConfigOption(stringInput("pantheonMajorGod", v))
			ec.note("set major god " + v)
		}
		if op.Minor != "" {
			v, e := pantheonValue("pantheonMinorGod", op.Minor)
			if e != nil {
				return e
			}
			b.Build.PantheonMinorGod = v
			b.SetConfigOption(stringInput("pantheonMinorGod", v))
			ec.note("set minor god " + v)
		}
	case "set_config", "unset_config":
		if op.Var == "" {
			return invalidArg(op.Op+" needs var.", nil)
		}
		o, e := resolveConfig(op.Var)
		if e != nil {
			return e
		}
		switch o.Var {
		case "bandit":
			return applyBuildOp(ec, buildOp{Op: "set_bandit", Bandit: fmt.Sprint(valueOr(op, o))})
		case "pantheonMajorGod":
			return applyBuildOp(ec, buildOp{Op: "set_pantheon", Major: fmt.Sprint(valueOr(op, o))})
		case "pantheonMinorGod":
			return applyBuildOp(ec, buildOp{Op: "set_pantheon", Minor: fmt.Sprint(valueOr(op, o))})
		}
		if op.Op == "unset_config" {
			b.RemoveConfigOption(o.Var)
			ec.note("unset_config " + o.Var)
			return nil
		}
		if op.Value == nil {
			return invalidArg("set_config needs value.", nil)
		}
		in, isDefault, e := configInput(o, op.Value)
		if e != nil {
			return e
		}
		if isDefault {
			b.RemoveConfigOption(o.Var)
		} else {
			b.SetConfigOption(in)
		}
		ec.note(fmt.Sprintf("set_config %s (%s) = %v", o.Var, o.cleanLabel(), op.Value))
	case "set_notes":
		b.SetNotes(op.Text)
		ec.note("set_notes")
	case "append_notes":
		notes := b.Notes
		if notes != "" {
			notes += "\n"
		}
		b.SetNotes(notes + op.Text)
		ec.note("append_notes")
	default:
		return invalidArg(fmt.Sprintf("Unknown op %q.", op.Op), buildOps)
	}
	return nil
}

// stringInput is the config input PoB's ConfigTab stores for bandit and pantheon, which the calculation reads over
// the build's own attributes.
func stringInput(name, v string) pob.Input {
	return pob.Input{Name: name, String: &v}
}

func valueOr(op buildOp, o configOption) any {
	if op.Op == "unset_config" || op.Value == nil {
		return o.defaultValue()
	}
	return op.Value
}
