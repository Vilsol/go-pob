package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	reClassOptional = regexp.MustCompile(`%[adws]\?`)
	reParenGroup    = regexp.MustCompile(`\([^)]*\)`)
)

func maximalExpand(k string) string {
	if reClassOptional.MatchString(k) {
		return ""
	}
	r := []rune(k)
	out := make([]rune, 0, len(r))
	for i := 0; i < len(r); {
		if i+1 < len(r) && r[i+1] == '?' && r[i] != '\n' && (i == 0 || r[i-1] != '%') {
			out = append(out, r[i])
			i += 2
			continue
		}
		out = append(out, r[i])
		i++
	}
	return string(out)
}

func optionalTokenCensus(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "REFUSING: usage: optional_token_census.py "+
			"<pinned ModParser.lua> <our mod_parser.go>")
		return 1
	}
	pinPath, oursPath := args[0], args[1]

	pinSrc, rc := readUTF8(pinPath)
	if rc != 0 {
		return rc
	}
	ourRaw, rc := readUTF8(oursPath)
	if rc != 0 {
		return rc
	}
	ourSrc := strings.ToLower(ourRaw)

	var keys []string
	for _, m := range rePinKey.FindAllStringSubmatch(pinSrc, -1) {
		keys = append(keys, m[1])
	}
	if len(keys) < 500 {
		fmt.Fprintf(os.Stderr, "REFUSING: only %d keys parsed from the pin; extraction is broken\n",
			len(keys))
		return 1
	}

	ctrlHit := maximalExpand("enemies ignited by you during f?l?a?s?k? ?effect take (%d+)%% " +
		"increased damage")
	if !strings.Contains(ctrlHit, "during flask effect") {
		fmt.Fprintln(os.Stderr, "REFUSING: the known optional-token case did not expand; the "+
			"expander is broken")
		return 1
	}
	if maximalExpand("monsters cannot block your attacks") != "monsters cannot block your attacks" {
		fmt.Fprintln(os.Stderr, "REFUSING: a key with no optional was altered by the expander")
		return 1
	}

	type hit struct{ core, key string }
	var hits []hit
	withOptionals := 0
	for _, k := range keys {
		if !strings.Contains(k, "?") {
			continue
		}
		withOptionals++
		m := maximalExpand(k)
		if m == "" {
			continue
		}
		core := strings.ToLower(rePctBs.ReplaceAllString(m, ""))
		core = reParenGroup.ReplaceAllString(core, "")
		core = strings.Join(strings.Fields(core), " ")
		if len([]rune(core)) < 12 {
			continue
		}
		if strings.Contains(ourSrc, core) {
			hits = append(hits, hit{core, k})
		}
	}

	fmt.Printf("pin keys %d   with optionals %d   HARDCODED HERE %d\n",
		len(keys), withOptionals, len(hits))
	fmt.Println()
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].core != hits[j].core {
			return hits[i].core < hits[j].core
		}
		return hits[i].key < hits[j].key
	})
	for _, h := range hits[:min(60, len(hits))] {
		fmt.Printf("  ours has literally: %s\n", truncRunes(h.core, 78))
		fmt.Printf("  pin  pattern      : %s\n", truncRunes(h.key, 78))
		fmt.Println()
	}
	return 0
}
