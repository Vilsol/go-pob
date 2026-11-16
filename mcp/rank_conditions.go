package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// checkCondition is a check-box config option of an assumption section and the condition its ifCond/ifEnemyCond names.
type checkCondition struct {
	Var   string `json:"var"`
	Type  string `json:"type"`
	Cond  string `json:"ifCond"`
	Enemy string `json:"ifEnemyCond"`
}

var assumptionChecks = sync.OnceValue(func() []checkCondition {
	var all []checkCondition
	_ = json.Unmarshal(configOptionsJSON, &all)
	var out []checkCondition
	for _, o := range all {
		if o.Type == "check" && (o.Cond != "" || o.Enemy != "") && slices.Contains(assumptionSections, configSectionOf()[o.Var]) &&
			!slices.ContainsFunc(out, func(x checkCondition) bool { return x.Var == o.Var }) {
			out = append(out, o)
		}
	}
	return out
})

// negatedUse reports whether a modifier the build has reads cond negated ("if you haven't Crit Recently").
func negatedUse(used []mod.Mod, cond string) bool {
	for _, m := range used {
		for _, tag := range m.Tags() {
			switch t := tag.(type) {
			case *mod.ConditionTag:
				if t.Negative && slices.Contains(t.VarList, cond) {
					return true
				}
			case *mod.ActorConditionTag:
				if t.Negative && slices.Contains(t.VariableList, cond) {
					return true
				}
			}
		}
	}
	return false
}

// assumedOff lists the build's unset combat-condition options whose condition a modifier reads negated: leaving them
// off assumes the favourable case, as forcing one on does for the others.
func assumedOff(b *pob.PathOfBuilding, env *calculator.Environment) []string {
	set := map[string]bool{}
	for _, in := range b.Config.Inputs {
		if in.Boolean != nil && *in.Boolean {
			set[in.Name] = true
		}
	}
	var out []string
	for _, o := range assumptionChecks() {
		if set[o.Var] {
			continue
		}
		if (o.Cond != "" && negatedUse(env.ConditionsUsed[o.Cond], o.Cond)) || (o.Enemy != "" && negatedUse(env.EnemyConditionsUsed[o.Enemy], o.Enemy)) {
			out = append(out, o.Var)
		}
	}
	return out
}

// questioned is b with each named forced condition unset, each named stage or stack count halved (the average over a
// build-up from none) and each named assumed-off condition turned on.
func questioned(b *pob.PathOfBuilding, forced, off []string) *pob.PathOfBuilding {
	c := cloneBuild(b)
	c.Config.Inputs = slices.Clone(c.Config.Inputs)
	for i, in := range c.Config.Inputs {
		if slices.Contains(forced, in.Name) && stageOptions()[in.Name] && in.Number != nil {
			half := math.Floor(*in.Number / 2)
			c.Config.Inputs[i].Number = &half
		}
	}
	c.Config.Inputs = slices.DeleteFunc(c.Config.Inputs, func(in pob.Input) bool {
		return slices.Contains(forced, in.Name) && (!stageOptions()[in.Name] || in.Number == nil || *in.Number == 0)
	})
	for _, name := range off {
		on := true
		c.Config.Inputs = append(c.Config.Inputs, pob.Input{Name: name, Boolean: &on})
	}
	return c
}

// conditionWords are the words of a forced check option's condition: "conditionEnemyLightningExposure" -> lightning,
// exposure; none for an option that is a recency, a charge use or a range assumption rather than an effect something
// inflicts.
func conditionWords(v string) []string {
	if strings.Contains(v, "Recently") {
		return nil
	}
	stripped := false
	for _, p := range []string{"conditionEnemy", "condition", "buff"} {
		if rest, ok := strings.CutPrefix(v, p); ok {
			v, stripped = rest, true
			break
		}
	}
	if !stripped || v == "" {
		return nil
	}
	var words []string
	start := 0
	for i := 1; i <= len(v); i++ {
		if i == len(v) || unicode.IsUpper(rune(v[i])) {
			if w := strings.ToLower(v[start:i]); w != "effect" {
				words = append(words, w)
			}
			start = i
		}
	}
	if len(words) == 0 || len(words[0]) < 5 {
		return nil
	}
	return words
}

var sourcePatterns = sync.Map{}

// sourcePattern matches a line that inflicts or grants the effect stem names, not one that only reads it ("while
// Fortified", "per Fortification").
func sourcePattern(stem string) *regexp.Regexp {
	if p, ok := sourcePatterns.Load(stem); ok {
		return p.(*regexp.Regexp)
	}
	s := regexp.QuoteMeta(stem) + `\w*`
	p := regexp.MustCompile(`(?i)(chance to (\w+ )?` + s + `|(hits?|always|are|you have( your maximum)?|inflict|gain) ` + s + `|\b` + s + ` (on|for \d|enemies|nearby))`)
	sourcePatterns.Store(stem, p)
	return p
}

// namesEffect reports whether line is a gem or skill name ("Fortify", "Intimidating Cry") whose first word is a form of
// word, not a compound that only starts like it ("Shockwave").
func namesEffect(word, line string) bool {
	f := strings.Fields(strings.ToLower(line))
	if len(f) == 0 || len(f) > 2 {
		return false
	}
	n := 0
	for n < len(f[0]) && n < len(word) && f[0][n] == word[n] {
		n++
	}
	return n >= 5 && n >= len(f[0])-3
}

// onYou ends a line that scales an effect on you ("increased Effect of Arcane Surge on you") rather than grants it.
var onYou = regexp.MustCompile(`(?i) on you\b`)

// sourceMatches reports whether line inflicts or grants the condition option v names: it mentions every word of the
// condition, and inflicts or names one of them, or is the gem named by all of them ("Arcane Surge").
func sourceMatches(v, line string) bool {
	words := conditionWords(v)
	if len(words) == 0 {
		return false
	}
	low := strings.ToLower(line)
	line = onYou.ReplaceAllString(line, "")
	hit := false
	for _, w := range words {
		stem := w[:min(len(w), 5)]
		if !strings.Contains(low, stem) {
			return false
		}
		hit = hit || sourcePattern(stem).MatchString(line) || (len(words) == 1 && namesEffect(w, line))
	}
	return hit || strings.EqualFold(strings.TrimSpace(line), strings.Join(words, " "))
}

// sourceLines are the texts a build's effects come from: allocated node stats, equipped item lines and enabled gems.
func sourceLines(b *pob.PathOfBuilding) map[string]bool {
	out := map[string]bool{}
	v := specTreeVersion(b)
	for _, id := range b.Build.PassiveNodes {
		if n, ok := treeNodeByID(v, id); ok {
			for _, st := range n.Stats {
				out[st] = true
			}
		}
	}
	for _, e := range equippedItems(b) {
		for _, l := range strings.Split(e.text, "\n") {
			out[strings.TrimSpace(l)] = true
		}
	}
	for _, g := range *groupsOf(b) {
		if !g.Enabled {
			continue
		}
		for _, gem := range g.Gems {
			if gem.Enabled {
				out[strings.TrimSuffix(gem.NameSpec, " Support")] = true
			}
		}
	}
	return out
}

// sourceFlags names the forced conditions whose source the change removes: with the condition forced on in config,
// losing what inflicts it costs nothing in any calculation.
func sourceFlags(forced []string, base, after map[string]bool) []string {
	var out []string
	for _, f := range forced {
		var lost []string
		for l := range base {
			if !after[l] && sourceMatches(f, l) {
				lost = append(lost, l)
			}
		}
		if len(lost) == 0 {
			continue
		}
		slices.Sort(lost)
		l := lost[0]
		if len(l) > 60 {
			l = l[:57] + "..."
		}
		out = append(out, fmt.Sprintf("config source: %s is forced on but this removes %q", f, l))
	}
	return out
}
