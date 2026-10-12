package main

import (
	"fmt"
	"os"
	"strings"
)

const ssgUsage = "usage: skill_stat_map_global_emit.py <dump.json> <out.go>"

const ssmgSkillFlagLine = "\t\tSkillFlag: %s,"

func skillStatMapGlobalEmit(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, ssgUsage)
		return 2
	}
	src, dst := args[0], args[1]

	d, reason := pyLoadJSONObject(src)
	if reason != "" {
		if strings.HasSuffix(reason, "is not a JSON object") {
			return ssgRefuse(fmt.Sprintf("%s is not a JSON object", src))
		}
		return ssgRefuse(strings.Replace(reason, src, fmt.Sprintf("the dump (%s)", src), 1))
	}
	for _, field := range []string{"entryCount", "modCount"} {
		if !ssmHas(d, field) {
			return ssgRefuse(fmt.Sprintf(
				"the dump carries no %s. Re-run oracle/skill_stat_map_global.lua.", pyRepr(field)))
		}
	}
	entries, ok := d["entries"].(map[string]any)
	if !ok {
		return ssgRefuse("the dump's 'entries' is not a JSON object")
	}

	for _, field := range []string{"order", "entries", "skillTypeNames"} {
		if !oeTruthy(d[field]) {
			return ssgRefuse(fmt.Sprintf(
				"the dump has no %s. Re-run oracle/skill_stat_map_global.lua.", pyRepr(field)))
		}
	}
	order, _ := d["order"].([]any)
	if len(order) != len(entries) {
		return ssgRefuse(fmt.Sprintf("the dump's order (%d) and entries (%d) disagree; one of "+
			"them lost a row.", len(order), len(entries)))
	}

	vocab, vreason := mcUseFlagTables(d)
	if vreason != "" {
		return mcRefuse(vreason)
	}
	skillTypes, _ := d["skillTypeNames"].(map[string]any)
	r := &ssmRun{
		vocab: vocab, skillTypes: skillTypes,
		omit: newPyCounter(), omitDetail: newPyListMap(), boolLiterals: map[string]int{},
	}

	body := []string{}
	keptMods := 0
	emptyEntries := []string{}
	for _, keyAny := range order {
		key := oeStr(keyAny)
		raw, ok := entries[key]
		if !ok {
			return ssgRefuse(fmt.Sprintf("the dump's 'order' names %s, which 'entries' does not "+
				"carry", pyRepr(key)))
		}
		e, ok := raw.(map[string]any)
		if !ok {
			return ssgRefuse(fmt.Sprintf("entry %s has no 'mods' list", pyRepr(key)))
		}
		rawMods, ok := e["mods"].([]any)
		if !ok {
			return ssgRefuse(fmt.Sprintf("entry %s has no 'mods' list", pyRepr(key)))
		}
		mods := []string{}
		for _, mAny := range rawMods {
			m, ok := mAny.(map[string]any)
			if !ok {
				r.omit.add("a mod is not a table [unverifiable: the dump shape is data-dependent]")
				continue
			}
			got, err := r.emitMod(m)
			if err == nil {
				mods = append(mods, got)
				keptMods++
				continue
			}
			var u *mcUnrepresentableError
			if !errorsAs(err, &u) {
				return mcRefuse(err.Error())
			}
			r.omit.add(u.reason)
			name := "<nameless>"
			if n, ok := m["name"]; ok {
				name = oeStr(n)
			}
			r.omitDetail.add(u.reason, key+"/"+name)
		}

		if comments, ok := e["comments"].([]any); ok {
			for _, c := range comments {
				s := oeStr(c)
				if len(s) >= 2 {
					s = s[2:]
				}
				body = append(body, "\t//"+s)
			}
		}

		if len(mods) == 0 {
			emptyEntries = append(emptyEntries, key)
		}

		body = append(body, "\t"+mcGoStr(key)+": {")
		switch {
		case len(mods) == 1:
			body = append(body, "\t\tMods: []mod.Mod{"+mods[0]+"},")
		case len(mods) > 1:
			body = append(body, "\t\tMods: []mod.Mod{")
			for _, m := range mods {
				body = append(body, "\t\t\t"+m+",")
			}
			body = append(body, "\t\t},")
		default:
			body = append(body, "\t\tMods: []mod.Mod{},")
		}
		scalars, _ := e["scalars"].(map[string]any)
		for _, kv := range [][2]string{{"value", "Value"}, {"mult", "Mult"},
			{"div", "Div"}, {"base", "Base"}} {
			if x, ok := scalars[kv[0]]; ok {
				u, err := mcUnwrap(x)
				if err != nil {
					return mcRefuse(err.Error())
				}
				body = append(body, fmt.Sprintf("\t\t%s: utils.Ptr(float64(%s)),", kv[1], mcGoNum(u)))
			}
		}
		if x, ok := scalars["skillFlag"]; ok {
			u, err := mcUnwrap(x)
			if err != nil {
				return mcRefuse(err.Error())
			}
			body = append(body, fmt.Sprintf(ssmgSkillFlagLine, mcGoStr(oeStr(u))))
		}
		for _, k := range oeSortedKeys(scalars) {
			if strings.HasPrefix(k, "__") {
				continue
			}
			switch k {
			case "value", "mult", "div", "base", "skillFlag":
			default:
				//go-only: the python original had NO such guard. It copied value/mult/div/base out
				return mcRefuse(fmt.Sprintf(
					"stat %s carries statMap scalar %s, which this emitter neither models nor "+
						"declares. Model it or declare it; emitting anyway is how skillFlag was "+
						"lost on both entries that had one.", pyRepr(key), pyRepr(k)))
			}
		}
		body = append(body, "\t},")
	}

	omittedTotal := r.omit.total()
	headerOmissions := []string{}
	for _, reason := range r.omit.mostCommon() {
		headerOmissions = append(headerOmissions,
			fmt.Sprintf("//   - %d %s", r.omit.counts[reason], reason))
	}
	omissions := strings.Join(headerOmissions, "\n")
	if omissions == "" {
		omissions = "//   (none)"
	}

	dupnote := ""
	if dups, ok := d["duplicateKeys"].([]any); ok && len(dups) > 0 {
		lines := make([]string, 0, len(dups))
		for _, k := range dups {
			lines = append(lines, "//   "+oeStr(k))
		}
		dupnote = "//\n// ⚠️ PoB declares these keys TWICE and a Lua table constructor assigns " +
			"duplicates in\n// order, so the LAST block wins. That is what is emitted here:\n" +
			strings.Join(lines, "\n")
	}

	out := fmt.Sprintf(ssgHeader, oeStr(sfeGet(d, "pobVersion", "unknown")),
		oeStr(d["entryCount"]), oeStr(d["modCount"]), keptMods, omittedTotal,
		omissions, dupnote) + strings.Join(body, "\n") + "\n}\n"
	if err := os.WriteFile(dst, []byte(out), 0o644); err != nil { //nolint:gosec // the original's mode
		return ssgRefuse(fmt.Sprintf("%s is not writable: %s", dst, errnoText(err)))
	}

	fmt.Fprintf(os.Stderr, "entries %s  mods %s -> kept %d, omitted %d\n",
		oeStr(d["entryCount"]), oeStr(d["modCount"]), keptMods, omittedTotal)
	r.reportBoolLiterals()
	for _, reason := range r.omit.mostCommon() {
		fmt.Fprintf(os.Stderr, "  %4d  %s\n", r.omit.counts[reason], reason)
		for _, x := range r.omitDetail.head(reason, 3) {
			fmt.Fprintf(os.Stderr, "          e.g. %s\n", x)
		}
	}
	if len(emptyEntries) > 0 {
		head := emptyEntries
		if len(head) > 8 {
			head = head[:8]
		}
		fmt.Fprintf(os.Stderr, "  %4d entries left with NO mods: %s\n",
			len(emptyEntries), strings.Join(head, ", "))
	}
	return 0
}

func ssgRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return 3
}
