package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

const ecdUsage = "usage: extract_calcs_damage.py <dumpsDir> <out.jsonl>"

var ecdOptionalTables = []string{"outputMainHand", "outputOffHand"}

func extractCalcsDamage(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, ecdUsage)
		return 1
	}
	dumpsDir, outPath := args[0], args[1]

	entries, err := os.ReadDir(dumpsDir)
	if err != nil {
		return eooRefuse(fmt.Sprintf("%s is not a readable directory: %s", dumpsDir, errnoText(err)))
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return eooRefuse(fmt.Sprintf("%s holds no .json dumps", dumpsDir))
	}

	out, err := os.Create(outPath) //nolint:gosec // an operator-supplied path, as in the original
	if err != nil {
		return eooRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
	}
	defer out.Close() //nolint:errcheck // the flush below is what can fail meaningfully
	w := bufio.NewWriter(out)

	written := 0
	for _, name := range names {
		dump, code := eooLoadDump(pyPathJoin(dumpsDir, name))
		if code != 0 {
			return code
		}
		rec := map[string]any{"build": strings.TrimSuffix(name, ".json") + ".xml"}
		if eooTruthy(dump["error"]) {
			rec["error"] = dump["error"]
			rec["output"] = map[string]any{}
		} else {
			rec["mainSkill"] = ""
			if v, ok := dump["mainSkill"]; ok && eooTruthy(v) {
				rec["mainSkill"] = v
			}
			rec["output"] = eooNumeric(dump["output"])
			for _, key := range ecdOptionalTables {
				if v, ok := dump[key]; ok && v != nil {
					rec[key] = eooNumeric(v)
				}
			}
		}
		var b strings.Builder
		pyDumps(rec, &b)
		if _, err := w.WriteString(b.String() + "\n"); err != nil {
			return eooRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
		}
		written++
	}
	if err := w.Flush(); err != nil {
		return eooRefuse(fmt.Sprintf("cannot write %s: %s", outPath, errnoText(err)))
	}
	fmt.Fprintf(os.Stderr, "calcs side-car: wrote %d records to %s\n", written, outPath)
	return 0
}
