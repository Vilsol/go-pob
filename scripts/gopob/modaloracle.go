package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const moUsage = "usage: modal_oracle.sh <dumps-base> <runs> <out.jsonl.gz> [workdir]"

var moArgNames = []string{"<dumps-base>", "<runs>", "<out.jsonl.gz>"}

func moFatal(code int, format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	return code
}

func modalOracle(argv []string) int {
	for i, name := range moArgNames {
		if len(argv) < i+1 {
			fmt.Fprintf(os.Stderr, "FATAL: missing %s\n%s\n", name, moUsage)
			return 2
		}
	}
	base, runsArg, out := argv[0], argv[1], argv[2]
	work := "/tmp/go-pob-modal-extract"
	if len(argv) > 3 {
		work = argv[3]
	}

	if !dpIsUint(runsArg) {
		return moFatal(2, "<runs> must be a positive integer, not '%s'", runsArg)
	}
	runs, err := strconv.Atoi(runsArg)
	if err != nil || runs < 1 {
		return moFatal(2, "<runs> must be at least 1, not %s", runsArg)
	}

	_ = os.MkdirAll(work, 0o750) //nolint:gosec // an operator-supplied workdir, as in the original

	type running struct {
		cmd *exec.Cmd
		log *os.File
	}
	var inputs []string
	var started []running
	for i := 1; i <= runs; i++ {
		d := fmt.Sprintf("%s-%d", base, i)
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() { //nolint:gosec // an operator-supplied path, as in the original
			return moFatal(2, "no dump dir %s", d)
		}
		j := pyPathJoin(work, fmt.Sprintf("run-%d.jsonl", i))
		if moStale(d, j) {
			fmt.Printf("extracting %s -> %s\n", d, j)
			cmd := exec.Command("nice", "-n", "15", "python3", //nolint:gosec,noctx // as in the original
				//fold-dead-ok: the difftest STUBS this path by design — it asserts which commands
				//fold-dead-ok: modal-oracle runs and with what argv, not what they compute, so a
				//fold-dead-ok: re-exec of the folded subcommand cannot be substituted and the
				//fold-dead-ok: harness becomes an integration test. RETIRED BY: modaloracle's
				//fold-dead-ok: difftest being rebuilt against the Go subcommands, or dropped as
				//fold-dead-ok: obsolete now that its original is deleted.
				pyPathJoin(moHere(), "oracle", "extract_output_oracle.py"), d, j)
			lf, err := os.Create(j + ".log") //nolint:gosec // an operator-supplied workdir path
			if err == nil {
				cmd.Stdout, cmd.Stderr = lf, lf
			}
			_ = cmd.Start()
			started = append(started, running{cmd, lf})
		} else {
			fmt.Printf("reusing %s\n", j)
		}
		inputs = append(inputs, j)
	}
	for i, r := range started {
		err := r.cmd.Wait()
		if r.log != nil {
			_ = r.log.Close()
		}
		if err != nil {
			return moFatal(4, "an extraction failed; see %s/run-%d.jsonl.log", work, i+1)
		}
	}

	plain := strings.TrimSuffix(out, ".gz")
	minfrac := os.Getenv("MINFRAC")
	if minfrac == "" {
		minfrac = "0.5"
	}
	args := append([]string{
		//fold-dead-ok: same as the extractor above — the difftest substitutes a stub here.
		pyPathJoin(moHere(), "oracle", "modal_blob.py"),
		"--min-fraction=" + minfrac, plain, out + ".report.txt",
	}, inputs...)
	blob := exec.Command("python3", args...) //nolint:gosec,noctx // as in the original
	blob.Stdout, blob.Stderr = os.Stdout, os.Stderr
	if err := blob.Run(); err != nil {
		return awExitStatus(err)
	}

	gz := exec.Command("gzip", "-9", "-n", "-f", plain) //nolint:gosec,noctx // as in the original
	gz.Stdout, gz.Stderr = os.Stdout, os.Stderr
	if err := gz.Run(); err != nil {
		return awExitStatus(err)
	}

	fmt.Printf("modal blob: %s\n", out)
	fmt.Printf("report:     %s.report.txt\n", out)
	return 0
}

func moSelf() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return os.Args[0]
}

func moHere() string {
	if v := os.Getenv("GOPOB_ROOT"); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := wd; ; {
		if fi, err := os.Stat(pyPathJoin(dir, "scripts", "modal_oracle.sh")); err == nil && !fi.IsDir() { //nolint:gosec // walking up from the CWD, as the original resolves from $0
			return dir
		}
		parent := pyPathStr(pyPathJoin(dir, ".."))
		if parent == dir || dir == "/" {
			return wd
		}
		dir = parent
	}
}

func moStale(dumpDir, jsonl string) bool {
	ji, err := os.Stat(jsonl) //nolint:gosec // an operator-supplied workdir path, as in the original
	if err != nil || ji.Size() == 0 {
		return true
	}
	di, err := os.Stat(dumpDir) //nolint:gosec // an operator-supplied path, as in the original
	if err != nil {
		return false
	}
	return di.ModTime().After(ji.ModTime())
}
