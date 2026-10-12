package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const spUsage = "usage: skill_parts.py <out.go>"

const spRefusalExit = 3

var spFiles = []string{"act_dex.lua", "act_int.lua", "act_str.lua", "glove.lua", "minion.lua",
	"other.lua", "spectre.lua", "sup_dex.lua", "sup_int.lua", "sup_str.lua"}

var (
	spSkillRE = regexp.MustCompile(`^skills\["([^"]+)"\]` + pySpaceClass + `*=`)
	spStrRE   = regexp.MustCompile(`^[\t ]+name` + pySpaceClass + `*=` + pySpaceClass +
		`*"((?:[^"\\]|\\\\.)*)"` + pySpaceClass + `*,?$`)
	spBoolRE = regexp.MustCompile(`^[\t ]+([A-Za-z][A-Za-z0-9]*)` + pySpaceClass + `*=` +
		pySpaceClass + `*(true|false)` + pySpaceClass + `*,?$`)
)

type spFlag struct {
	name  string
	value bool
}

type spPart struct {
	name  string
	flags []spFlag
}

type spSkill struct {
	id, loc string
	parts   []spPart
}

type spRefusalError struct{ msg string }

func (e *spRefusalError) Error() string { return e.msg }

func spRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return spRefusalExit
}

func spParse(path, rel string) ([]spSkill, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // POB_DIR-relative, as in the original
	if err != nil {
		return nil, &spRefusalError{fmt.Sprintf("%s is not readable: %s", rel, errnoText(err))}
	}
	body := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(body, "\n")

	at := func(i int) (string, error) {
		if i >= len(lines) {
			return "", &spRefusalError{fmt.Sprintf(
				"%s: a parts block runs past the end of the file", rel)}
		}
		return lines[i], nil
	}
	isClose := func(s string) bool { return s == "}," || s == "}" }

	out := []spSkill{}
	skill := ""
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if m := spSkillRE.FindStringSubmatch(line); m != nil {
			skill = m[1]
			continue
		}
		if skill == "" || pyStrip(line) != "parts = {" || !strings.HasPrefix(line, "\tp") {
			continue
		}
		loc := fmt.Sprintf("%s:%d", rel, i+1)
		parts := []spPart{}
		i++
		for {
			cur, err := at(i)
			if err != nil {
				return nil, err
			}
			if isClose(pyStrip(cur)) {
				break
			}
			if pyStrip(cur) != "{" {
				return nil, &spRefusalError{fmt.Sprintf("%s:%d: expected a part constructor, got %s",
					rel, i+1, pyRepr(cur))}
			}
			i++
			name, found, flags := "", false, []spFlag{}
			for {
				cur, err := at(i)
				if err != nil {
					return nil, err
				}
				if isClose(pyStrip(cur)) {
					break
				}
				stripped := pyRStrip(cur)
				if ms := spStrRE.FindStringSubmatch(stripped); ms != nil {
					name, found = ms[1], true
				} else if mb := spBoolRE.FindStringSubmatch(stripped); mb != nil {
					flags = append(flags, spFlag{name: mb[1], value: mb[2] == "true"})
				} else if pyStrip(cur) != "" {
					return nil, &spRefusalError{fmt.Sprintf("%s:%d: unrecognised line in a part: %s",
						rel, i+1, pyRepr(cur))}
				}
				i++
			}
			if !found {
				return nil, &spRefusalError{fmt.Sprintf("%s:%d: a part declares no name", rel, i+1)}
			}
			parts = append(parts, spPart{name: name, flags: flags})
			i++
		}
		out = append(out, spSkill{id: skill, loc: loc, parts: parts})
		skill = ""
	}
	return out, nil
}

var spTypeDoc = strings.ReplaceAll(spTypeDocTemplate, "@", "`")

const spTypeDocTemplate = `
// SkillPart is one entry of PoB's hand-written per-skill @parts@ table
// (src/Data/Skills/*.lua). PoB keys a part's flag overrides by skill-flag name in the same
// table as @name@, and CalcActiveSkill.lua:206-212 walks it setting the flag for @true@ and
// CLEARING it for @false@; Flags keeps the two apart because a Go map read cannot.
type SkillPart struct {
	Name  string
	Flags map[string]bool
}

// SkillParts mirrors @grantedEffect.parts@, keyed by granted-effect id. It exists for the
// same reason as SkillBaseFlags: the table is hand-written PoB data that go-pob-data does
// not ship. PoB reads it at src/Modules/CalcActiveSkill.lua:196 to pick the part the build
// selected (@<Gem skillPart="N">@), apply that part's flag overrides, and set the
// @multiPart@ flag. Flag names are PoB's raw strings — data/raw cannot import calculator,
// whose SkillFlag is a string type holding exactly these values.
//
// Generated from PoB v2.18.1 by oracle/skill_parts.py.
`

func skillParts(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, spUsage)
		return 2
	}
	outPath := args[0]

	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		pobDir = "~/PathOfBuilding"
	}
	skillsDir := filepath.Join(pbdRealpath(pobDir), "src", "Data", "Skills")

	all := []spSkill{}
	for _, fn := range spFiles {
		rel := fmt.Sprintf("src/Data/%s/%s", filepath.Base(skillsDir), fn)
		got, err := spParse(filepath.Join(skillsDir, fn), rel)
		if err != nil {
			return spRefuse(err.Error())
		}
		all = append(all, got...)
	}
	sort.SliceStable(all, func(a, b int) bool { return all[a].id < all[b].id })

	nparts, nflags := 0, 0
	for _, s := range all {
		nparts += len(s.parts)
		for _, p := range s.parts {
			nflags += len(p.flags)
		}
	}
	fmt.Fprintf(os.Stderr, "%d skills, %d parts, %d flag overrides\n", len(all), nparts, nflags)

	w := []string{"package raw\n", spTypeDoc, "var SkillParts = map[string][]SkillPart{"}
	for _, s := range all {
		w = append(w, "\t// "+s.loc, "\t"+spGoStr(s.id)+": {")
		for _, p := range s.parts {
			if len(p.flags) == 0 {
				w = append(w, "\t\t{Name: "+spGoStr(p.name)+"},")
				continue
			}
			fl := make([]string, 0, len(p.flags))
			for _, f := range p.flags {
				fl = append(fl, fmt.Sprintf("%s: %t", spGoStr(f.name), f.value))
			}
			w = append(w, fmt.Sprintf("\t\t{Name: %s, Flags: map[string]bool{%s}},",
				spGoStr(p.name), strings.Join(fl, ", ")))
		}
		w = append(w, "\t},")
	}
	w = append(w, "}")

	body := strings.Join(w, "\n") + "\n"
	if err := os.WriteFile(outPath, []byte(body), 0o644); err != nil { //nolint:gosec // the original's mode
		return spRefuse(fmt.Sprintf("%s is not writable: %s", outPath, errnoText(err)))
	}
	return 0
}

func spGoStr(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}
