package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

const pddUsage = "usage: pob_dump_diff.py <dirA> <dirB> [--verbose]"

var pddBlobFields = map[string]bool{
	"output": true, "outputMainHand": true, "outputOffHand": true, "enemyOutput": true,
	"minionOutput": true, "minionOutputMainHand": true, "minionOutputOffHand": true,
}

var pddDerivedFields = map[string]bool{"modDB": true}

var pddPtr = regexp.MustCompile(`\b(function|table|thread|userdata|cdata): 0x[0-9a-fA-F]+`)

var pddPtrHits int

func pddRefuse(msg string) int {
	fmt.Printf("REFUSING: %s\n", msg)
	fmt.Print("\nThis is NOT a verdict. Nothing was compared.\n")
	return 2
}

func pddRefusePlain(msg string) int {
	fmt.Printf("REFUSING: %s\n", msg)
	return 2
}

func pddOSError(err error, path string) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return err.Error()
	}
	text := errno.Error()
	if text != "" {
		text = strings.ToUpper(text[:1]) + text[1:]
	}
	return fmt.Sprintf("[Errno %d] %s: %s", int(errno), text, pyRepr(path))
}

func pddNormalise(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	pddPtrHits += len(pddPtr.FindAllStringIndex(s, -1))
	return pddPtr.ReplaceAllString(s, "${1}: <ptr>")
}

func pddCanonRepr(v any) string {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = "(" + pyRepr(k) + ", " + pddCanonRepr(t[k]) + ")"
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []any:
		inner := make([]string, len(t))
		for i, e := range t {
			inner[i] = pddCanonRepr(e)
		}
		sort.Strings(inner)
		quoted := make([]string, len(inner))
		for i, s := range inner {
			quoted[i] = pyRepr(s)
		}
		return "[" + strings.Join(quoted, ", ") + "]"
	default:
		return pyReprValue(pddNormalise(v))
	}
}

func pddIsNum(v any) bool {
	_, ok := v.(json.Number)
	return ok
}

func pddWalk(a, b any, path string, out *[]string) {
	if pddIsNum(a) && pddIsNum(b) {
		fa, ea := a.(json.Number).Float64()
		fb, eb := b.(json.Number).Float64()
		if ea != nil || eb != nil || fa != fb {
			*out = append(*out, fmt.Sprintf("%s: %s vs %s", path, pyReprValue(a), pyReprValue(b)))
		}
		return
	}
	if pyTypeName(a) != pyTypeName(b) {
		*out = append(*out, fmt.Sprintf("%s: TYPE %s vs %s", path, pyTypeName(a), pyTypeName(b)))
		return
	}
	switch ta := a.(type) {
	case map[string]any:
		tb, _ := b.(map[string]any)
		for _, k := range pddKeyUnion(ta, tb) {
			_, inA := ta[k]
			_, inB := tb[k]
			switch {
			case !inA:
				*out = append(*out, fmt.Sprintf("%s.%s: ONLY-B", path, k))
			case !inB:
				*out = append(*out, fmt.Sprintf("%s.%s: ONLY-A", path, k))
			default:
				pddWalk(ta[k], tb[k], fmt.Sprintf("%s.%s", path, k), out)
			}
		}
	case []any:
		tb, _ := b.([]any)
		if pddCanonRepr(a) != pddCanonRepr(b) {
			*out = append(*out, fmt.Sprintf("%s: %d vs %d entries, multisets differ",
				path, len(ta), len(tb)))
		}
	default:
		na, nb := pddNormalise(a), pddNormalise(b)
		if !pddScalarEqual(na, nb) {
			*out = append(*out, fmt.Sprintf("%s: %s vs %s", path, pyReprValue(a), pyReprValue(b)))
		}
	}
}

func pddScalarEqual(a, b any) bool {
	switch ta := a.(type) {
	case nil:
		return b == nil
	case string:
		tb, ok := b.(string)
		return ok && ta == tb
	case bool:
		tb, ok := b.(bool)
		return ok && ta == tb
	}
	return pyReprValue(a) == pyReprValue(b)
}

func pddKeyUnion(a, b map[string]any) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func pddLoad(dir, name string) (map[string]any, int) {
	m, reason := pyLoadJSONObject(pyPathJoin(dir, name))
	if reason != "" {
		return nil, pddRefuse(reason)
	}
	return m, 0
}

func pddListDumps(dir string) ([]string, int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, pddRefusePlain(pddOSError(err, dir))
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, 0
}

func pobDumpDiff(args []string) int {
	if len(args) < 2 {
		fmt.Println(pddUsage)
		return 2
	}
	dirA, dirB := args[0], args[1]
	verbose := false
	for _, a := range args {
		if a == "--verbose" {
			verbose = true
		}
	}

	fa, rc := pddListDumps(dirA)
	if rc != 0 {
		return rc
	}
	fb, rc := pddListDumps(dirB)
	if rc != 0 {
		return rc
	}
	if len(fa) == 0 {
		return pddRefusePlain(fmt.Sprintf("%s holds no .json dumps -- nothing was compared", dirA))
	}
	if !pddSameSet(fa, fb) {
		fmt.Println("REFUSING: the two directories hold different build sets")
		fmt.Printf("  only in A: %s\n", pddOnlyIn(fa, fb))
		fmt.Printf("  only in B: %s\n", pddOnlyIn(fb, fa))
		return 2
	}

	order := []string{"BLOB", "DERIVED", "OTHER", "breakdown"}
	buckets := map[string][]string{}
	for _, k := range order {
		buckets[k] = nil
	}

	for _, name := range fa {
		a, code := pddLoad(dirA, name)
		if code != 0 {
			return code
		}
		b, code := pddLoad(dirB, name)
		if code != 0 {
			return code
		}

		ta, tb := eooTreeSources(a), eooTreeSources(b)
		if ta != tb {
			buckets["BLOB"] = append(buckets["BLOB"], fmt.Sprintf(
				"%s: treeSources %d vs %d -- THIS CAN MOVE THE 184-BUILD EXCLUSION SET",
				name, ta, tb))
		}

		for _, k := range pddKeyUnion(a, b) {
			which := "OTHER"
			switch {
			case k == "breakdown":
				which = "breakdown"
			case pddBlobFields[k]:
				which = "BLOB"
			case pddDerivedFields[k]:
				which = "DERIVED"
			}
			va, inA := a[k]
			vb, inB := b[k]
			switch {
			case !inA:
				buckets[which] = append(buckets[which], fmt.Sprintf("%s.%s: ONLY-B", name, k))
			case !inB:
				buckets[which] = append(buckets[which], fmt.Sprintf("%s.%s: ONLY-A", name, k))
			default:
				var x, y any = va, vb
				if which == "BLOB" {
					x, y = eooNumeric(va), eooNumeric(vb)
				}
				rows := buckets[which]
				pddWalk(x, y, fmt.Sprintf("%s.%s", name, k), &rows)
				buckets[which] = rows
			}
		}
	}

	fmt.Printf("compared %d builds\n", len(fa))
	fmt.Printf("  %d Lua pointer stringification(s) normalised to <ptr> -- see normalise()\n", pddPtrHits)
	fmt.Printf("  A %s\n", dirA)
	fmt.Printf("  B %s\n", dirB)
	fmt.Println()
	notes := map[string]string{
		"BLOB":      "seven output tables + recomputed treeSources",
		"DERIVED":   "modDB structurally; informational, never fatal",
		"OTHER":     "censuses only; compare against YOUR measured floor",
		"breakdown": "not extracted into the blob",
	}
	for _, which := range order {
		fmt.Printf("  %-10s %6d   (%s)\n", which, len(buckets[which]), notes[which])
	}

	for _, which := range order[:3] {
		rows := buckets[which]
		shown := rows
		if !verbose && len(rows) > 30 {
			shown = rows[:30]
		}
		for _, d := range shown {
			fmt.Printf("    [%s] %s\n", which, d)
		}
		if !verbose && len(rows) > 30 {
			fmt.Printf("    [%s] ... %d more (--verbose for all)\n", which, len(rows)-30)
		}
	}

	fatal := buckets["BLOB"]
	fmt.Println()
	if len(fatal) > 0 {
		fmt.Println("NOT OUTPUT-NEUTRAL: the oracle blob would change. Do not keep this change " +
			"without a deliberate, measured regeneration.")
		return 1
	}
	fmt.Println("Blob-neutral: no oracle field and no treeSources moved. Check DERIVED and OTHER " +
		"against the floor you measured on an UNMODIFIED A/B before calling this clean.")
	return 0
}

func pddSameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func pddOnlyIn(a, b []string) string {
	inB := make(map[string]bool, len(b))
	for _, s := range b {
		inB[s] = true
	}
	var only []string
	for _, s := range a {
		if !inB[s] {
			only = append(only, s)
		}
	}
	sort.Strings(only)
	if len(only) == 0 {
		return "-"
	}
	if len(only) > 10 {
		only = only[:10]
	}
	return pyReprValue(only)
}
