package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const chunkUsage = "usage: chunk_init.py <file.go> <VarName> [--per N] [--check]"

var reVarDecl = `(?m)^var %s = (.*?)\{$`

func chunkRefuse(msg string) int {
	fmt.Fprintln(os.Stderr, msg)
	return root3
}

const root3 = 3

type chunkArgs struct {
	path, varName string
	per           int
	check         bool
}

func chunkParseArgs(argv []string) (chunkArgs, int) {
	a := chunkArgs{per: 120}
	var pos []string
	for i := 0; i < len(argv); {
		switch arg := argv[i]; {
		case arg == "--check":
			a.check = true
			i++
		case arg == "--per":
			if i+1 >= len(argv) {
				return a, chunkRefuse("chunk-init: --per needs a value\n" + chunkUsage)
			}
			if !regexp.MustCompile(`^[1-9][0-9]*$`).MatchString(argv[i+1]) {
				return a, chunkRefuse("chunk-init: --per must be a positive integer, not " +
					argv[i+1] + "\n" + chunkUsage)
			}
			a.per, _ = strconv.Atoi(argv[i+1])
			i += 2
		case strings.HasPrefix(arg, "-"):
			return a, chunkRefuse("chunk-init: unknown argument " + pyRepr(arg) + "\n" + chunkUsage)
		default:
			pos = append(pos, arg)
			i++
		}
	}
	if len(pos) != 2 {
		return a, chunkRefuse(fmt.Sprintf("chunk-init: need exactly <file.go> and <VarName>, "+
			"got %d\n%s", len(pos), chunkUsage))
	}
	a.path, a.varName = pos[0], pos[1]
	return a, 0
}

type span struct{ start, end int }

func scanTopLevel(body []rune) []span {
	var spans []span
	depth, i, n := 0, 0, len(body)
	entryStart := -1
	for i < n {
		c := body[i]
		if entryStart < 0 && !unicode.IsSpace(c) {
			entryStart = i
		}
		if c == '/' && i+1 < n && body[i+1] == '/' {
			if j := indexRune(body, i, '\n'); j < 0 {
				i = n
			} else {
				i = j
			}
			continue
		}
		if c == '/' && i+1 < n && body[i+1] == '*' {
			if j := indexRunes(body, i+2, []rune("*/")); j < 0 {
				i = n
			} else {
				i = j + 2
			}
			continue
		}
		if c == '`' {
			if j := indexRune(body, i+1, '`'); j < 0 {
				i = n
			} else {
				i = j + 1
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote := c
			j := i + 1
			for j < n {
				if body[j] == '\\' {
					j += 2
					continue
				}
				if body[j] == quote {
					break
				}
				j++
			}
			i = j + 1
			continue
		}
		switch {
		case c == '{' || c == '(' || c == '[':
			depth++
		case c == '}' || c == ')' || c == ']':
			depth--
		case c == ',' && depth == 0:
			if entryStart >= 0 {
				spans = append(spans, span{entryStart, i + 1})
				entryStart = -1
			}
			i++
			continue
		}
		i++
	}
	if entryStart >= 0 && strings.TrimSpace(string(body[entryStart:])) != "" {
		spans = append(spans, span{entryStart, n})
	}
	return spans
}

func indexRune(r []rune, from int, want rune) int {
	for i := from; i < len(r); i++ {
		if r[i] == want {
			return i
		}
	}
	return -1
}

func indexRunes(r []rune, from int, want []rune) int {
	for i := from; i+len(want) <= len(r); i++ {
		if string(r[i:i+len(want)]) == string(want) {
			return i
		}
	}
	return -1
}

func valueType(mapType string) (string, int) {
	if !strings.HasPrefix(mapType, "map[") {
		return "", chunkRefuse("REFUSING: " + mapType +
			" is not a map type; this script only handles maps")
	}
	r := []rune(mapType)
	depth := 0
	for i := 3; i < len(r); i++ {
		if r[i] == '[' {
			depth++
		} else if r[i] == ']' {
			depth--
			if depth == 0 {
				return string(r[i+1:]), 0
			}
		}
	}
	return "", chunkRefuse("REFUSING: cannot split key type out of " + mapType)
}

func splitKey(entry string) (string, string, int) {
	r := []rune(entry)
	depth, i, n := 0, 0, len(r)
	for i < n {
		c := r[i]
		if c == '`' {
			if j := indexRune(r, i+1, '`'); j < 0 {
				i = n
			} else {
				i = j + 1
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote := c
			j := i + 1
			for j < n {
				if r[j] == '\\' {
					j += 2
					continue
				}
				if r[j] == quote {
					break
				}
				j++
			}
			i = j + 1
			continue
		}
		switch {
		case c == '{' || c == '(' || c == '[':
			depth++
		case c == '}' || c == ')' || c == ']':
			depth--
		case c == ':' && depth == 0:
			return strings.TrimSpace(string(r[:i])), strings.TrimSpace(string(r[i+1:])), 0
		}
		i++
	}
	return "", "", chunkRefuse("REFUSING: no top-level colon in entry: " + truncRunes(entry, 60))
}

func findLiteral(src string, varName string) (int, int, int, string, int) {
	re := regexp.MustCompile(fmt.Sprintf(reVarDecl, regexp.QuoteMeta(varName)))
	loc := re.FindStringSubmatchIndex(src)
	if loc == nil {
		return 0, 0, 0, "", chunkRefuse("REFUSING: no `var " + varName + " = ...{` declaration found")
	}
	mapType := src[loc[2]:loc[3]]
	declStart := utf8.RuneCountInString(src[:loc[0]])
	brace := utf8.RuneCountInString(src[:loc[1]]) - 1

	r := []rune(src)
	depth, i, n := 0, brace, len(r)
	for i < n {
		c := r[i]
		if c == '/' && i+1 < n && r[i+1] == '/' {
			if j := indexRune(r, i, '\n'); j < 0 {
				i = n
			} else {
				i = j
			}
			continue
		}
		if c == '/' && i+1 < n && r[i+1] == '*' {
			if j := indexRunes(r, i+2, []rune("*/")); j < 0 {
				i = n
			} else {
				i = j + 2
			}
			continue
		}
		if c == '`' {
			if j := indexRune(r, i+1, '`'); j < 0 {
				i = n
			} else {
				i = j + 1
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote := c
			j := i + 1
			for j < n {
				if r[j] == '\\' {
					j += 2
					continue
				}
				if r[j] == quote {
					break
				}
				j++
			}
			i = j + 1
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return declStart, brace, i, mapType, 0
			}
		}
		i++
	}
	return 0, 0, 0, "", chunkRefuse("REFUSING: unbalanced braces in " + varName)
}

var reLeadComment = regexp.MustCompile(`^\s*(//[^\n]*)\n`)

func chunkInit(argv []string) int {
	args, rc := chunkParseArgs(argv)
	if rc != 0 {
		return rc
	}
	b, err := os.ReadFile(args.path) //nolint:gosec // an operator-supplied path
	if err != nil {
		return chunkRefuse(fmt.Sprintf("chunk-init: cannot read %s: %s", args.path, errnoText(err)))
	}
	src := string(b)
	declStart, brace, closeIdx, mapType, rc := findLiteral(src, args.varName)
	if rc != 0 {
		return rc
	}
	r := []rune(src)
	body := r[brace+1 : closeIdx]
	spans := scanTopLevel(body)
	if len(spans) == 0 {
		return chunkRefuse("REFUSING: " + args.varName + " has no top-level entries")
	}

	vtype, rc := valueType(mapType)
	if rc != 0 {
		return rc
	}
	lower := strings.ToLower(args.varName[:1]) + args.varName[1:]
	var parts [][]span
	for i := 0; i < len(spans); i += args.per {
		parts = append(parts, spans[i:min(i+args.per, len(spans))])
	}

	var out strings.Builder
	out.WriteString(string(r[:declStart]))
	fmt.Fprintf(&out, "var %s = build%s()\n\n", args.varName, args.varName)
	fmt.Fprintf(&out, "func build%s() %s {\n\tm := make(%s, %d)\n", args.varName, mapType, mapType, len(spans))
	for idx := range parts {
		fmt.Fprintf(&out, "\t%sPart%d(m)\n", lower, idx)
	}
	out.WriteString("\treturn m\n}\n")

	for idx, part := range parts {
		fmt.Fprintf(&out, "\nfunc %sPart%d(m %s) {\n", lower, idx, mapType)
		for _, sp := range part {
			entry := strings.TrimRight(string(body[sp.start:sp.end]), " \t\n\r\v\f")
			entry = strings.TrimSuffix(entry, ",")
			var lead []string
			for {
				m2 := reLeadComment.FindStringSubmatchIndex(entry)
				if m2 == nil {
					break
				}
				lead = append(lead, entry[m2[2]:m2[3]])
				entry = entry[m2[1]:]
			}
			key, value, rc := splitKey(entry)
			if rc != 0 {
				return rc
			}
			if strings.HasPrefix(value, "{") {
				value = vtype + value
			}
			for _, line := range lead {
				fmt.Fprintf(&out, "\t%s\n", line)
			}
			fmt.Fprintf(&out, "\tm[%s] = %s\n", key, value)
		}
		out.WriteString("}\n")
	}
	out.WriteString(string(r[closeIdx+1:]))
	newSrc := out.String()

	if args.check {
		if newSrc != src {
			return 1
		}
		return 0
	}
	if err := os.WriteFile(args.path, []byte(newSrc), 0o600); err != nil { //nolint:gosec // operator path
		return chunkRefuse(fmt.Sprintf("chunk-init: cannot write %s: %s", args.path, errnoText(err)))
	}
	fmt.Printf("%s: %s -> %d entries in %d chunk(s)\n", args.path, args.varName, len(spans), len(parts))
	return 0
}
