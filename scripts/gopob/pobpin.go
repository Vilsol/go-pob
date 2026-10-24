package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const (
	ppOK = iota
	ppMismatch
	ppNoCheckout
	ppNotACheckout
	ppUnreadable
	ppNoRecord
)

var ppMarkers = []string{"src/Modules/Main.lua", "src/Launch.lua", "src/HeadlessWrapper.lua"}

type ppRefuseError struct {
	code int
	msg  string
}

func (e *ppRefuseError) Error() string { return e.msg }

func ppRefuseErr(code int, format string, args ...any) error {
	return &ppRefuseError{code, fmt.Sprintf(format, args...)}
}

func ppRepoRoot() string {
	if v := os.Getenv("GOPOB_ROOT"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for _, anchor := range [][]string{{"scripts", "pob_pin.json"}, {".git"}} {
		for dir := wd; ; {
			if _, err := os.Stat(pyPathJoin(append([]string{dir}, anchor...)...)); err == nil { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return wd
}

// ppRecordName is the pin record for the game being checked: PoE1 unless --game poe2.
var ppRecordName = "pob_pin.json"

func ppRecordPath() string { return pyPathJoin(ppRepoRoot(), "scripts", ppRecordName) }

func ppTrackedFiles(pob string, scope []string) ([]string, error) {
	args := append([]string{"-C", pob, "ls-files", "-z", "--"}, scope...)
	cmd := exec.Command("git", args...) //nolint:gosec,noctx // the original is a bare subprocess.run
	var errb strings.Builder
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			return nil, ppRefuseErr(ppUnreadable,
				"git is not installed, so the tracked file set cannot be listed")
		}
		return nil, ppRefuseErr(ppUnreadable, "`git ls-files` failed in %s: %s",
			pob, strings.TrimSpace(errb.String()))
	}
	files := []string{}
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files, nil
}

func ppContentHash(root string, files []string) (string, error) {
	return ppContentHashTampered(root, files, "")
}

// ppContentHashTampered hashes as if one more byte were appended to the file `tamper`.
func ppContentHashTampered(root string, files []string, tamper string) (string, error) {
	h := sha256.New()
	for _, rel := range files {
		full := pyPathJoin(root, rel)
		blob, err := os.ReadFile(full) //nolint:gosec // a git-tracked path inside the checkout
		if err != nil {
			return "", ppRefuseErr(ppUnreadable, "cannot read %s: %s", full, ppOSErr(full, err))
		}
		if rel == tamper {
			blob = append(blob, ' ')
		}
		h.Write([]byte(rel))
		h.Write([]byte(fmt.Sprintf("\x00%d\x00", len(blob))))
		h.Write(blob)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ppOSErr(path string, err error) string {
	var errno syscall.Errno
	n := 0
	if errors.As(err, &errno) {
		n = int(errno)
	}
	return fmt.Sprintf("[Errno %d] %s: %s", n, errnoText(err), pyRepr(path))
}

func ppGitIdentity(pob string) (any, any, bool) {
	run := func(args ...string) any {
		cmd := exec.Command("git", append([]string{"-C", pob}, args...)...) //nolint:gosec,noctx // as in the original
		out, err := cmd.Output()
		if err != nil {
			return nil
		}
		return strings.TrimSpace(string(out))
	}
	status := run("status", "--porcelain", "--untracked-files=no")
	s, _ := status.(string)
	return run("describe", "--tags"), run("rev-parse", "HEAD"), s != ""
}

func ppLocate(pob string) (string, error) {
	if pob == "" {
		return "", ppRefuseErr(ppNoCheckout, "POB_DIR is empty and no default was given")
	}
	pob = pyExpandUser(pob)
	if abs, err := filepath.Abs(pob); err == nil {
		pob = abs
	}
	st, err := os.Stat(pob) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	if err != nil {
		return "", ppRefuseErr(ppNoCheckout,
			"%s does not exist -- set POB_DIR to the PoB source checkout", pob)
	}
	if !st.IsDir() {
		return "", ppRefuseErr(ppNoCheckout, "%s is not a directory", pob)
	}
	missing := []string{}
	for _, m := range ppMarkers {
		if fi, err := os.Stat(pyPathJoin(pob, m)); err != nil || fi.IsDir() { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		hint := ""
		if fi, err := os.Stat(pyPathJoin(pob, "Launch.lua")); err == nil && !fi.IsDir() { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
			hint = "\n  This looks like an INSTALLED PoB (Launch.lua at the top level, no src/)" +
				"\n  rather than a source checkout. That is the ~/PoB/PoE1 decoy the pin" +
				"\n  guards against: it is a DIFFERENT, NEWER version of the program."
		}
		return "", ppRefuseErr(ppNotACheckout, "%s is not a PoB source checkout; missing %s%s",
			pob, strings.Join(missing, ", "), hint)
	}
	if _, err := os.Stat(pyPathJoin(pob, ".git")); err != nil { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
		return "", ppRefuseErr(ppUnreadable,
			"%s has no .git, so its tracked file set cannot be determined", pob)
	}
	return pob, nil
}

func ppMeasure(pob string, scope []string) ([]string, string, error) {
	files, err := ppTrackedFiles(pob, scope)
	if err != nil {
		return nil, "", err
	}
	if len(files) == 0 {
		return nil, "", ppRefuseErr(ppUnreadable,
			"git tracks no files under %s in %s -- refusing to hash an empty set",
			strings.Join(scope, ", "), pob)
	}
	digest, err := ppContentHash(pob, files)
	return files, digest, err
}

func ppLoadRecord() (map[string]any, error) {
	path := ppRecordPath()
	fi, err := os.Stat(path) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	if err != nil || fi.IsDir() {
		if err == nil && fi.IsDir() {
			return nil, ppRefuseErr(ppNoRecord, "%s is missing -- the pin has no recorded identity", path)
		}
		return nil, ppRefuseErr(ppNoRecord, "%s is missing -- the pin has no recorded identity", path)
	}
	rec, reason := ppReadRecord(path)
	if reason != "" {
		return nil, ppRefuseErr(ppNoRecord, "%s", reason)
	}
	for _, key := range []string{"scope", "content_sha256", "file_count"} {
		if _, ok := rec[key]; !ok {
			return nil, ppRefuseErr(ppNoRecord, "%s has no %s field", path, pyRepr(key))
		}
	}
	return rec, nil
}

func ppReadRecord(path string) (map[string]any, string) {
	raw, err := os.ReadFile(path) //nolint:gosec // the record beside the script
	if err != nil {
		return nil, fmt.Sprintf("%s is unreadable: %s", path, errnoText(err))
	}
	m, reason := pyLoadJSONObjectBytes(raw, path)
	if reason != "" {
		if strings.HasSuffix(reason, "is not a JSON object") {
			return nil, fmt.Sprintf("%s is not a JSON object", path)
		}
		return nil, fmt.Sprintf("%s is malformed (not valid JSON)", path)
	}
	return m, ""
}

func ppScope(rec map[string]any) []string {
	a, _ := rec["scope"].([]any)
	out := make([]string, 0, len(a))
	for _, x := range a {
		out = append(out, oeStr(x))
	}
	return out
}

func ppReindexCodegraph(pob string, always bool) {
	cg := pyPathJoin(pob, ".codegraph")
	if fi, err := os.Stat(cg); err != nil || !fi.IsDir() { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
		return
	}
	db := pyPathJoin(cg, "codegraph.db")
	if dbfi, err := os.Stat(db); !always && err == nil && !dbfi.IsDir() { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
		var newest float64
		for _, scope := range []string{"src", "runtime/lua"} {
			_ = filepath.Walk(pyPathJoin(pob, scope), func(_ string, info os.FileInfo, werr error) error { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
				if werr != nil || info == nil || info.IsDir() {
					return nil //nolint:nilerr // os.walk's onerror=None, and the original try/excepts each stat
				}
				if m := float64(info.ModTime().UnixNano()) / 1e9; m > newest {
					newest = m
				}
				return nil
			})
		}
		if newest <= float64(dbfi.ModTime().UnixNano())/1e9 {
			return
		}
	}
	cmd := exec.Command("codegraph", "index", pob) //nolint:gosec,noctx // as in the original
	var errb strings.Builder
	cmd.Stderr = &errb
	if err := cmd.Run(); err == nil {
		fmt.Printf("  codegraph: reindexed %s (the pinned tree moved past its index)\n", pob)
		return
	}
	first := "?"
	for _, l := range strings.Split(strings.TrimSpace(errb.String()), "\n") {
		if strings.TrimSpace(l) != "" {
			first = l
			break
		}
	}
	fmt.Printf("  codegraph: NOT reindexed (%s). Tools may get stale line numbers; run "+
		"`codegraph index %s`.\n", first, pob)
}

func ppDoCheck(pob string) (int, error) {
	rec, err := ppLoadRecord()
	if err != nil {
		return 0, err
	}
	scope := ppScope(rec)
	files, digest, err := ppMeasure(pob, scope)
	if err != nil {
		return 0, err
	}
	describe, sha, dirty := ppGitIdentity(pob)
	ppReindexCodegraph(pob, false)

	fmt.Println("PoB pin check")
	fmt.Printf("  checkout   %s\n", pob)
	fmt.Printf("  scope      %s\n", strings.Join(scope, ", "))
	fmt.Printf("  tracked    %d files (recorded %s)\n", len(files), oeStr(rec["file_count"]))
	fmt.Printf("  describe   %s (recorded %s)\n", oeStr(describe), oeStr(mdGet(rec, "describe", nil)))
	fmt.Printf("  HEAD       %s\n", oeStr(sha))
	if dirty {
		fmt.Println("  NOTE       the checkout has uncommitted changes to TRACKED files;")
		fmt.Println("             they are inside content_sha256, so this is only a warning if the")
		fmt.Println("             hash still matches -- but the tree is then not reproducible from")
		fmt.Println("             its commits alone.")
	}
	if digest != oeStr(rec["content_sha256"]) {
		fmt.Println("\nFAIL: the PoB checkout is NOT the tree the oracle was built against.")
		fmt.Printf("  expected content_sha256 %s\n", oeStr(rec["content_sha256"]))
		fmt.Printf("  got                     %s\n", digest)
		if n, ok := gemsInt(rec["file_count"]); !ok || n != len(files) {
			fmt.Printf("  file count differs too: %d vs recorded %s\n",
				len(files), oeStr(rec["file_count"]))
		}
		fmt.Println("\n  Every accuracy figure, oracleMatchedFloor, and every divergence pin was")
		fmt.Println("  measured against the recorded tree. Do not regenerate the oracle from this")
		fmt.Println("  checkout.")
		return ppMismatch, nil
	}
	fmt.Println("\nOK: content matches the recorded pin.")
	recSha := oeStr(mdGet(rec, "sha", nil))
	if s, ok := sha.(string); ok && s != "" && recSha != "" && recSha != "None" && s != recSha {
		fmt.Println("  (HEAD differs from the recorded SHA but content is identical -- expected after")
		fmt.Println("   replaying the fork commits onto a fresh clone. Content is the assertion.)")
	}
	return ppOK, nil
}

func ppDoRecord(pob string) (int, error) {
	rec := map[string]any{}
	path := ppRecordPath()
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() { //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
		got, reason := ppReadRecord(path)
		if reason != "" {
			return 0, ppRefuseErr(ppNoRecord, "%s", reason)
		}
		rec = got
	}
	scope := ppScope(rec)
	if len(scope) == 0 {
		scope = []string{"src", "runtime/lua"}
	}
	files, digest, err := ppMeasure(pob, scope)
	if err != nil {
		return 0, err
	}
	describe, sha, dirty := ppGitIdentity(pob)
	rec["scope"] = mdAnyList(scope)
	rec["content_sha256"] = digest
	rec["file_count"] = len(files)
	rec["describe"] = describe
	rec["sha"] = sha
	rec["dirty_when_recorded"] = dirty

	body, merr := json.MarshalIndent(rec, "", "  ")
	if merr != nil {
		return 0, ppRefuseErr(ppUnreadable, "cannot render the record: %s", merr)
	}
	if werr := os.WriteFile(path, append(body, '\n'), 0o644); werr != nil { //nolint:gosec // the original's mode
		return 0, ppRefuseErr(ppUnreadable, "cannot write %s: %s", path, errnoText(werr))
	}
	fmt.Printf("recorded %d files, content_sha256=%s (%s)\n", len(files), digest, oeStr(describe))
	ppReindexCodegraph(pob, true)
	return ppOK, nil
}

func pobPin(args []string) int {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		pobDir = pyExpandUser("~/PathOfBuilding")
	}
	game := ""
	for i, a := range args {
		if a == "--game" && i+1 < len(args) {
			game = args[i+1]
		}
	}
	switch game {
	case "", "poe1":
	case "poe2":
		ppRecordName = "pob_pin_poe2.json"
		if pobDir = os.Getenv("POB2_DIR"); pobDir == "" {
			pobDir = pyExpandUser("~/" + pcDefaultPoB2Rel)
		}
	default:
		fmt.Printf("REFUSING (NO-RECORD): --game %s is not poe1 or poe2\n", pyRepr(game))
		return ppNoRecord
	}
	for _, a := range args {
		if a == "--selftest" {
			if game == "" {
				return ppSelftest("")
			}
			return ppSelftest(pobDir)
		}
	}
	record := false
	for _, a := range args {
		if a == "--record" {
			record = true
		}
	}
	pob, err := ppLocate(pobDir)
	code := 0
	if err == nil {
		if record {
			code, err = ppDoRecord(pob)
		} else {
			code, err = ppDoCheck(pob)
		}
	}
	if err != nil {
		var r *ppRefuseError
		if !errorsAs(err, &r) {
			r = &ppRefuseError{ppUnreadable, err.Error()}
		}
		name := map[int]string{
			ppNoCheckout: "NO-CHECKOUT", ppNotACheckout: "NOT-A-CHECKOUT",
			ppUnreadable: "UNREADABLE", ppNoRecord: "NO-RECORD",
		}[r.code]
		fmt.Printf("REFUSING (%s): %s\n", name, r.msg)
		fmt.Println("\nThis is NOT a pass. The pin could not be verified at all.")
		return r.code
	}
	return code
}

// ppSelftest proves the checks can fail; given a checkout (--game), it also proves that
// checkout's own record catches a one-byte change to one of its tracked files.
func ppSelftest(pin string) int {
	bad, total := 0, 9
	report := func(name string, ok bool, want, got string) {
		if !ok {
			bad++
		}
		verdict := "ok"
		if !ok {
			verdict = "FAIL"
		}
		fmt.Printf("  %-58s %s   want=%s got=%s\n", name, verdict, want, got)
	}
	codeOf := func(pob string) int {
		if _, err := ppLocate(pob); err != nil {
			var r *ppRefuseError
			if errorsAs(err, &r) {
				return r.code
			}
			return ppUnreadable
		}
		return ppOK
	}
	tmp, err := os.MkdirTemp("", "pobpin-")
	if err != nil {
		//go-only: tempfile.TemporaryDirectory raises OSError; reaching it needs a broken TMPDIR.
		fmt.Printf("REFUSING (UNREADABLE): selftest sandbox: %s\n", errnoText(err))
		return ppUnreadable
	}
	defer func() { _ = os.RemoveAll(tmp) }() //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original

	nope := pyPathJoin(tmp, "nope")
	report("a path that does not exist", codeOf(nope) == ppNoCheckout,
		"NO-CHECKOUT", strconv.Itoa(codeOf(nope)))

	f := pyPathJoin(tmp, "afile")
	_ = os.WriteFile(f, nil, 0o600) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	report("a file where a directory was expected", codeOf(f) == ppNoCheckout,
		"NO-CHECKOUT", strconv.Itoa(codeOf(f)))

	decoy := pyPathJoin(tmp, "PoE1")
	_ = os.MkdirAll(pyPathJoin(decoy, "Modules"), 0o750)          //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	_ = os.WriteFile(pyPathJoin(decoy, "Launch.lua"), nil, 0o600) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	report("an INSTALLED PoB (the ~/PoB/PoE1 decoy shape)", codeOf(decoy) == ppNotACheckout,
		"NOT-A-CHECKOUT", strconv.Itoa(codeOf(decoy)))

	nogit := pyPathJoin(tmp, "nogit")
	_ = os.MkdirAll(pyPathJoin(nogit, "src", "Modules"), 0o750) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	for _, m := range ppMarkers {
		_ = os.MkdirAll(filepath.Dir(pyPathJoin(nogit, m)), 0o750) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
		_ = os.WriteFile(pyPathJoin(nogit, m), nil, 0o600)         //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	}
	report("a source tree with no .git", codeOf(nogit) == ppUnreadable,
		"UNREADABLE", strconv.Itoa(codeOf(nogit)))

	h := pyPathJoin(tmp, "h")
	_ = os.MkdirAll(h, 0o750)                                  //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	_ = os.WriteFile(pyPathJoin(h, "a"), []byte("one"), 0o600) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	_ = os.WriteFile(pyPathJoin(h, "b"), []byte("two"), 0o600) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	base, _ := ppContentHash(h, []string{"a", "b"})
	again, _ := ppContentHash(h, []string{"a", "b"})
	report("the same files hash the same", again == base, "same", "same")
	_ = os.WriteFile(pyPathJoin(h, "b"), []byte("TWO"), 0o600) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	changed, _ := ppContentHash(h, []string{"a", "b"})
	report("a one-file content change moves the hash", changed != base, "differs", "differs")
	dropped, _ := ppContentHash(h, []string{"a"})
	report("dropping a file moves the hash", dropped != base, "differs", "differs")
	_ = os.WriteFile(pyPathJoin(h, "a"), []byte("onetwo"), 0o600) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	_ = os.WriteFile(pyPathJoin(h, "b"), nil, 0o600)              //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	joined, _ := ppContentHash(h, []string{"a", "b"})
	report("concatenation cannot alias a two-file set", joined != base, "differs", "differs")

	empty := pyPathJoin(tmp, "empty")
	_ = os.MkdirAll(empty, 0o750) //nolint:gosec // POB_DIR is the operator's own checkout path, as in the original
	_, _, merr := ppMeasure(empty, []string{"src"})
	emptyCode := ppOK
	var r *ppRefuseError
	if errorsAs(merr, &r) {
		emptyCode = r.code
	}
	report("an empty tracked set refuses rather than hashing nothing", emptyCode == ppUnreadable,
		"UNREADABLE", ppCodeName(emptyCode))

	if pin != "" {
		total += 2
		got, want, tampered := ppSelftestPin(pin)
		report("the recorded checkout matches its record", got == want, "matches", ppShort(got))
		report("one tampered byte in a tracked file moves the hash", tampered != "" && tampered != want,
			"differs", ppShort(tampered))
	}

	fmt.Printf("\nSELFTEST: %d of %d checks behave as declared\n", total-bad, total)
	if bad > 0 {
		return 1
	}
	return ppOK
}

func ppCodeName(code int) string {
	return map[int]string{
		ppOK: "OK", ppMismatch: "MISMATCH", ppNoCheckout: "NO-CHECKOUT",
		ppNotACheckout: "NOT-A-CHECKOUT", ppUnreadable: "UNREADABLE", ppNoRecord: "NO-RECORD",
	}[code]
}

func ppSelftestPin(pin string) (string, string, string) {
	pob, err := ppLocate(pin)
	if err != nil {
		return err.Error(), "", ""
	}
	rec, err := ppLoadRecord()
	if err != nil {
		return err.Error(), "", ""
	}
	files, got, err := ppMeasure(pob, ppScope(rec))
	if err != nil {
		return err.Error(), "", ""
	}
	tampered, err := ppContentHashTampered(pob, files, files[len(files)/2])
	if err != nil {
		return got, oeStr(rec["content_sha256"]), ""
	}
	return got, oeStr(rec["content_sha256"]), tampered
}

func ppShort(s string) string {
	if len(s) > 16 {
		return s[:16]
	}
	return s
}
