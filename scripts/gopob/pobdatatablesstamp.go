package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	pdtCannotAnswer = 3
	pdtPinMismatch  = 4
)

func pdtRefuse(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "REFUSING: "+format+"\n", args...)
	return pdtCannotAnswer
}

func pdtJoin(dir, name string) string {
	d := pyPathStr(dir)
	switch {
	case d == ".":
		return name
	case strings.HasSuffix(d, "/"):
		return d + name
	}
	return d + "/" + name
}

func pdtUnstampedIn(directory string) ([]string, int) {
	ents, err := os.ReadDir(directory)
	if err != nil {
		ents = nil
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]string, 0, len(names))
	for _, n := range names {
		candidate := pdtJoin(directory, n)
		doc, code := pdtReadReference(candidate)
		if code != 0 {
			return nil, code
		}
		if !pdtTruthy(doc["pobPin"]) {
			out = append(out, candidate)
		}
	}
	return out, 0
}

func pdtReadReference(path string) (map[string]any, int) {
	doc, reason := pyLoadJSONObject(path)
	if reason != "" {
		return nil, pdtRefuse("%s", reason)
	}
	return doc, 0
}

func pdtTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	}
	if f, err := strconv.ParseFloat(oeStr(v), 64); err == nil {
		return f != 0
	}
	return true
}

func pdtPercentD(v any) string {
	if f, err := strconv.ParseFloat(oeStr(v), 64); err == nil {
		return strconv.FormatInt(int64(f), 10)
	}
	return oeStr(v)
}

func pobDataTablesStamp(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: pob_data_tables_stamp.py <reference.json|reference-dir>")
		return 2
	}
	path := pyPathStr(args[0])

	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		pending, code := pdtUnstampedIn(path)
		if code != 0 {
			return code
		}
		if len(pending) == 0 {
			fmt.Fprintf(os.Stderr,
				"nothing to stamp in %s -- every reference already carries a pin\n", path)
			return 0
		}
		if len(pending) > 1 {
			names := make([]string, 0, len(pending))
			for _, p := range pending {
				names = append(names, pyPathName(p))
			}
			return pdtRefuse("%d unstamped references in %s: %s.\n"+
				"  Stamping them all would give several files the same pin, and the gate\n"+
				"  refuses ambiguity by design. Remove the stale ones.",
				len(pending), path, strings.Join(names, ", "))
		}
		path = pending[0]
	}

	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		pobDir = pyExpandUser("~/PathOfBuilding")
	}
	pob, err := ppLocate(pobDir)
	var rec map[string]any
	var files []string
	var digest string
	if err == nil {
		rec, err = ppLoadRecord()
	}
	if err == nil {
		files, digest, err = ppMeasure(pob, ppScope(rec))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: cannot establish the pin identity: %s\n", err)
		fmt.Fprint(os.Stderr,
			"  Without it the reference cannot say which checkout it came from, and the gate\n"+
				"  would have to take it on trust — the exact thing the stamp removes.\n")
		return pdtCannotAnswer
	}

	recorded, isStr := rec["content_sha256"].(string)
	if !isStr || digest != recorded {
		fmt.Fprintln(os.Stderr,
			"REFUSING: the checkout this reference was dumped from is NOT the pinned PoB.")
		fmt.Fprintf(os.Stderr, "  measured content_sha256 %s (%d files)\n", digest, len(files))
		fmt.Fprintf(os.Stderr, "  recorded content_sha256 %s (%s files)\n",
			oeStr(rec["content_sha256"]), pdtPercentD(rec["file_count"]))
		fmt.Fprint(os.Stderr,
			"  Run `task pob-pin` for the per-file breakdown. Re-dump from the pinned checkout,\n"+
				"  or re-record the pin deliberately — but do not stamp this one.\n")
		return pdtPinMismatch
	}

	dump, code := pdtReadReference(path)
	if code != 0 {
		return code
	}

	if !pdtTruthy(dump["tables"]) {
		return pdtRefuse("%s declares no tables. Stamping an empty reference would give it the\n"+
			"  authority of the pin while passing every comparison trivially.", path)
	}

	describe, sha, dirty := ppGitIdentity(pob)
	dump["pobPin"] = map[string]any{
		"content_sha256":    digest,
		"file_count":        len(files),
		"scope":             rec["scope"],
		"describe_advisory": describe,
		"sha_advisory":      sha,
		"dirty_advisory":    dirty,
	}

	var b strings.Builder
	pyDumpsIndent(dump, 1, 0, &b)
	b.WriteByte('\n')
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil { //nolint:gosec // an operator-supplied reference path, as in the original
		return pdtRefuse("cannot write %s: %s", path, errnoText(err))
	}
	fmt.Fprintf(os.Stderr, "stamped %s with content_sha256=%s (%d files, %s)\n",
		pyPathName(path), digest, len(files), oeStr(describe))
	return 0
}
