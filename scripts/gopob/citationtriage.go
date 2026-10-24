package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	reLuaRef  = regexp.MustCompile(`^(src/[^\s:]+\.lua):([1-9][0-9]*)(?:-([1-9][0-9]*))?$`)
	reGoFunc  = regexp.MustCompile(`^func (\([^)]*\) )?(\w+)`)
	rePosInt  = regexp.MustCompile(`^[1-9][0-9]*$`)
	reNonNegI = regexp.MustCompile(`^[0-9]+$`)
)

func citationTriage(args []string) int {
	pob := expandUser(pobPinDir())
	flags := os.Getenv("FLAGS")
	if flags == "" {
		flags = "/tmp/flags.txt"
	}

	before, rc := countArg(args, 0, "BEFORE", 3)
	if rc != 0 {
		return rc
	}
	after, rc := countArg(args, 1, "AFTER", 3)
	if rc != 0 {
		return rc
	}

	if _, err := os.Stat(flags); err != nil { //nolint:gosec // an operator-supplied path, as the original's os.path.exists
		fmt.Fprintf(os.Stderr, "REFUSING: no flag list at %s\n", flags)
		return 1
	}
	flagSrc, rc := readTextRefusing(flags)
	if rc != 0 {
		return rc
	}

	cache := map[string][]string{}
	n := 0
	for i, line := range strings.Split(flagSrc, "\n") {
		lineno := i + 1
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		goref, luaref := f[0], f[1]
		reason := strings.Join(f[2:], " ")
		gofile, goline, ok := splitRef(goref)
		if !ok {
			fmt.Fprintf(os.Stderr, "REFUSING: %s line %d: %s is not a file:line reference\n",
				flags, lineno, goref)
			return 1
		}
		m := reLuaRef.FindStringSubmatch(luaref)
		if m == nil {
			continue
		}
		path := m[1]
		start, _ := strconv.Atoi(m[2])
		end := start
		if m[3] != "" {
			end, _ = strconv.Atoi(m[3])
		}
		src, ok := cache[path]
		if !ok {
			full := filepath.Join(pob, path)
			if _, err := os.Stat(full); err != nil { //nolint:gosec // POB_DIR joined with a path from the operator's flag list
				fmt.Printf("!! %s  %s  NO SUCH FILE\n", goref, luaref)
				continue
			}
			text, rc := readTextRefusing(full)
			if rc != 0 {
				return rc
			}
			src = strings.Split(text, "\n")
			cache[path] = src
		}
		if start > len(src) || end > len(src) {
			fmt.Fprintf(os.Stderr, "REFUSING: %s is past the end of %s (%d lines)\n",
				luaref, path, len(src))
			return 1
		}
		sym, marker := goSymbol(gofile, goline)
		n++
		fmt.Printf("\n[%2d] %s   func %s\n", n, goref, sym)
		fmt.Printf("     %s\n", marker)
		fmt.Printf("     %s\n", reason)
		for i := max(1, start-before); i < start; i++ {
			fmt.Printf("       %6d - %s\n", i, src[i-1])
		}
		fmt.Printf("     >%6d | %s\n", start, src[start-1])
		if end != start {
			if end-start > 1 {
				fmt.Printf("       %6s ...  %d line(s) ...\n", "", end-start-1)
			}
			fmt.Printf("     <%6d | %s\n", end, src[end-1])
		}
		for i := end + 1; i < min(end+1+after, len(src)+1); i++ {
			fmt.Printf("       %6d + %s\n", i, src[i-1])
		}
	}
	fmt.Printf("\n%d flagged range(s)\n", n)
	return 0
}

func goSymbol(gofile string, goline int) (string, string) {
	b, err := os.ReadFile(gofile) //nolint:gosec // a path from the operator's own flag list
	if err != nil || !utf8.Valid(b) {
		return "?", "?"
	}
	src := strings.Split(string(b), "\n")
	marker := "?"
	if goline-1 < len(src) {
		marker = strings.TrimSpace(src[goline-1])
	}
	for i := goline - 1; i < min(goline+40, len(src)); i++ {
		if m := reGoFunc.FindStringSubmatch(src[i]); m != nil {
			return m[2], marker
		}
	}
	return "(block marker, no func below)", marker
}

func splitRef(ref string) (string, int, bool) {
	i := strings.LastIndex(ref, ":")
	if i <= 0 {
		return "", 0, false
	}
	if !rePosInt.MatchString(ref[i+1:]) {
		return "", 0, false
	}
	n, err := strconv.Atoi(ref[i+1:])
	return ref[:i], n, err == nil
}

func countArg(args []string, i int, name string, def int) (int, int) {
	if len(args) <= i {
		return def, 0
	}
	if !reNonNegI.MatchString(args[i]) {
		fmt.Fprintf(os.Stderr, "REFUSING: %s must be a non-negative integer, not %s\n", name, args[i])
		return 0, 1
	}
	n, _ := strconv.Atoi(args[i])
	return n, 0
}

func expandUser(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	return pyExpandUser(p)
}

func readTextRefusing(path string) (string, int) {
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
