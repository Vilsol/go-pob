package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const mkUsage = "usage: mk_drop_cases.py <census.tsv> <out.json> [top]"

var reMkInt = regexp.MustCompile(`^-?[0-9]+$`)

func mkDropCases(argv []string) int {
	if len(argv) < 2 {
		fmt.Fprintln(os.Stderr, "REFUSING: "+mkUsage)
		return 1
	}
	tsvPath, outPath := argv[0], argv[1]
	top := 400
	if len(argv) > 2 {
		if !reNonNegI.MatchString(argv[2]) {
			fmt.Fprintf(os.Stderr, "REFUSING: top must be a non-negative integer, not %s\n", argv[2])
			return 1
		}
		top, _ = strconv.Atoi(argv[2])
	}

	b, err := os.ReadFile(tsvPath) //nolint:gosec // an operator-supplied path
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is not readable: %s\n", tsvPath, errnoText(err))
		return 1
	}
	r := csv.NewReader(strings.NewReader(string(b)))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	recs, err := r.ReadAll()
	if err != nil || len(recs) == 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: census TSV is empty")
		return 1
	}
	header := recs[0]
	idx := map[string]int{}
	for i, h := range header {
		idx[h] = i
	}
	need := []string{"builds", "occurrences", "kinds", "normalised", "sample_line", "leftover_extra"}
	for _, k := range need {
		if _, ok := idx[k]; !ok {
			fmt.Fprintf(os.Stderr, "REFUSING: header is %s, expected %s\n",
				pyList(header, len(header)), pyList(sortedCopy(need), len(need)))
			return 1
		}
	}
	rows := recs[1:]
	if len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "REFUSING: census TSV is empty")
		return 1
	}
	for i, row := range rows {
		v := ""
		if idx["builds"] < len(row) {
			v = strings.TrimSpace(row[idx["builds"]])
		}
		if !reMkInt.MatchString(v) {
			fmt.Fprintf(os.Stderr, "REFUSING: %s line %d: builds=%s is not an integer\n",
				tsvPath, i+2, pyRepr(v))
			return 1
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, _ := strconv.Atoi(strings.TrimSpace(rows[i][idx["builds"]]))
		b, _ := strconv.Atoi(strings.TrimSpace(rows[j][idx["builds"]]))
		return a > b
	})
	if top < len(rows) {
		rows = rows[:top]
	}

	type mkCase struct{ name, raw string }
	var cases []mkCase
	for i, row := range rows {
		line := ""
		if idx["sample_line"] < len(row) {
			line = strings.TrimSpace(row[idx["sample_line"]])
		}
		if line == "" {
			continue
		}
		raw := "Rarity: RARE\nProbe Item\nIron Ring\n--------\n" + line
		cases = append(cases, mkCase{
			name: fmt.Sprintf("drop%04d|%s|%s", i, strings.TrimSpace(row[idx["builds"]]),
				truncRunes(line, 80)),
			raw: raw,
		})
	}

	var out strings.Builder
	out.WriteString("[")
	for i, c := range cases {
		if i > 0 {
			out.WriteString(",")
		}
		out.WriteString("\n{\n")
		fmt.Fprintf(&out, "%s: %s,\n", pyJSONStringLiteral("name"), pyJSONStringLiteral(c.name))
		fmt.Fprintf(&out, "%s: %s\n", pyJSONStringLiteral("raw"), pyJSONStringLiteral(c.raw))
		out.WriteString("}")
	}
	if len(cases) > 0 {
		out.WriteString("\n")
	}
	out.WriteString("]")
	if err := os.WriteFile(outPath, []byte(out.String()), 0o600); err != nil { //nolint:gosec // operator path
		fmt.Fprintf(os.Stderr, "REFUSING: cannot write %s: %s\n", outPath, errnoText(err))
		return 1
	}
	fmt.Printf("wrote %s: %d cases from the top %d shapes by builds\n", outPath, len(cases), top)
	return 0
}

func sortedCopy(xs []string) []string {
	out := append([]string{}, xs...)
	sort.Strings(out)
	return out
}

func pyJSONStringLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
