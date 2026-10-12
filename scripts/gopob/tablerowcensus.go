package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	rePinKey = regexp.MustCompile(`(?m)^\s*\["([^"]+)"\]\s*=`)

	reCaptures = regexp.MustCompile(`\(\\d\+\)|\(%d\+\)|\(%a\+\)|\(%w\+\)|\(\\w\+\)|\(\[\\d\\.\]\+\)|\(%-?%d+%.?%d*\)`)
	reAnyGroup = regexp.MustCompile(`\([^)]*\)`)
	rePctBs    = regexp.MustCompile(`[%\\]`)
	reMetachar = regexp.MustCompile(`[\[\]\^\$\?\*\+]`)
	reNonWord  = regexp.MustCompile(`[^a-z0-9# ]+`)
)

func normKey(k string) string {
	s := strings.ToLower(k)
	s = reCaptures.ReplaceAllString(s, "#")
	s = reAnyGroup.ReplaceAllString(s, "#")
	s = strings.ReplaceAll(s, "%%", "%")
	s = strings.ReplaceAll(s, `\%`, "%")
	s = rePctBs.ReplaceAllString(s, "")
	s = reMetachar.ReplaceAllString(s, "")
	s = reNonWord.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func familyOf(n string) string {
	var words []string
	for _, w := range strings.Fields(n) {
		if w != "#" {
			words = append(words, w)
		}
	}
	if len(words) > 4 {
		words = words[:4]
	}
	return strings.Join(words, " ")
}

type famRow struct {
	miss, total, have int
	family            string
	examples          []string
}

func tableRowCensus(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "REFUSING: usage: table_row_census.py <pinned ModParser.lua> "+
			"<our mod_parser.go> [min_family_size]")
		return 1
	}
	pinPath, oursPath := args[0], args[1]
	minFam := 3
	if len(args) > 2 {
		if !reAllDigits.MatchString(args[2]) {
			fmt.Fprintf(os.Stderr,
				"REFUSING: min_family_size must be a non-negative integer, not %s\n", args[2])
			return 1
		}
		minFam, _ = strconv.Atoi(args[2])
	}

	pinSrc, rc := readUTF8(pinPath)
	if rc != 0 {
		return rc
	}
	ourSrc, rc := readUTF8(oursPath)
	if rc != 0 {
		return rc
	}

	pinKeys := map[string]bool{}
	for _, m := range rePinKey.FindAllStringSubmatch(pinSrc, -1) {
		pinKeys[normKey(m[1])] = true
	}
	delete(pinKeys, "")
	ourNorm := normKey(ourSrc)

	if len(pinKeys) < 500 {
		fmt.Fprintf(os.Stderr, "REFUSING: only %d keys parsed from the pin; the extraction is "+
			"broken and every family would look missing\n", len(pinKeys))
		return 1
	}
	have := normKey(`with at least 40 dexterity in radius, dual strike has (\d+)% increased ` +
		`attack speed while wielding a claw`)
	absent := normKey("zz not a real modifier line zz")
	if !strings.Contains(ourNorm, have) {
		fmt.Fprintln(os.Stderr, "REFUSING: a row this parser demonstrably carries was not found; "+
			"the containment test is broken and every family would look missing")
		return 1
	}
	if strings.Contains(ourNorm, absent) {
		fmt.Fprintln(os.Stderr, "REFUSING: a nonsense row was 'found'; the containment test "+
			"matches anything")
		return 1
	}

	famPin := map[string]map[string]bool{}
	famMissing := map[string]map[string]bool{}
	totalMissing := 0
	for k := range pinKeys {
		f := familyOf(k)
		if famPin[f] == nil {
			famPin[f] = map[string]bool{}
		}
		famPin[f][k] = true
		if !strings.Contains(ourNorm, k) {
			if famMissing[f] == nil {
				famMissing[f] = map[string]bool{}
			}
			famMissing[f][k] = true
		}
	}
	for _, v := range famMissing {
		totalMissing += len(v)
	}

	fmt.Printf("pin keys %d   pin keys absent here %d\n", len(pinKeys), totalMissing)
	fmt.Println()
	fmt.Printf("%5s %5s %5s  FAMILY  (only families with >= %d missing, and at least one row "+
		"ALREADY ported -- that is the partial-port signal)\n", "MISS", "PIN", "HAVE", minFam)

	var rows []famRow
	for f, missing := range famMissing {
		h := len(famPin[f]) - len(missing)
		if len(missing) >= minFam && h >= 1 {
			ex := make([]string, 0, len(missing))
			for k := range missing {
				ex = append(ex, k)
			}
			sort.Strings(ex)
			rows = append(rows, famRow{len(missing), len(famPin[f]), h, f, ex})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch {
		case a.miss != b.miss:
			return a.miss > b.miss
		case a.total != b.total:
			return a.total > b.total
		case a.have != b.have:
			return a.have > b.have
		default:
			return a.family > b.family
		}
	})

	for _, r := range headRows(rows, 40) {
		fmt.Printf("%5d %5d %5d  %s\n", r.miss, r.total, r.have, r.family)
		for _, e := range r.examples[:min(3, len(r.examples))] {
			fmt.Printf("%17s    - %s\n", "", truncRunes(e, 100))
		}
	}
	return 0
}

func headRows(rows []famRow, n int) []famRow {
	if n > len(rows) {
		n = len(rows)
	}
	return rows[:n]
}

func readUTF8(path string) (string, int) {
	b, err := os.ReadFile(path) //nolint:gosec // an operator-supplied path
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is not readable: %s\n", path, errnoText(err))
		return "", 1
	}
	if !utf8.Valid(b) {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is not valid UTF-8\n", path)
		return "", 1
	}
	return string(b), 0
}
