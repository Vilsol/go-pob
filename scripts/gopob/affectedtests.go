package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

type atPackage struct {
	ImportPath   string
	Dir          string
	Deps         []string
	TestImports  []string
	XTestImports []string
	GoFiles      []string
	TestGoFiles  []string
	XTestGoFiles []string
	Error        *struct{ Err string }
}

type atPick struct {
	pkg    *atPackage
	reason string
}

type atSelection struct {
	mapped   map[string]string
	notes    map[string]string
	picked   []atPick
	allPkgs  int
	wholeMod bool
}

func atRefuse(msg string) int {
	fmt.Fprintln(os.Stderr, "REFUSING: affected-tests: "+msg)
	return root.RefusalCode
}

func atGit(repo string, args ...string) ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // git with this tool's own arguments and BASE
	cmd.Dir = repo
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}

func atListPackages(repo string) ([]*atPackage, error) {
	cmd := exec.CommandContext(context.Background(), "go", "list", "-e", "-json=ImportPath,Dir,Deps,TestImports,XTestImports,GoFiles,TestGoFiles,XTestGoFiles,Error", "./...")
	cmd.Dir = repo
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var pkgs []*atPackage
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		p := &atPackage{}
		if err := dec.Decode(p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("go list output: %w", err)
		}
		if p.Error != nil {
			return nil, fmt.Errorf("go list: %s: %s", p.ImportPath, p.Error.Err)
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("go list %s listed no packages", "./...")
	}
	return pkgs, nil
}

func atIgnoredByGo(rel string) bool {
	for _, part := range strings.Split(filepath.Dir(rel), string(filepath.Separator)) {
		if part == "testdata" || strings.HasPrefix(part, "_") || strings.HasPrefix(part, ".") && part != "." {
			return true
		}
	}
	return false
}

func atSelect(repo string, pkgs []*atPackage, changed []string) atSelection {
	byDir := map[string]*atPackage{}
	byPath := map[string]*atPackage{}
	for _, p := range pkgs {
		byDir[p.Dir] = p
		byPath[p.ImportPath] = p
	}
	sel := atSelection{mapped: map[string]string{}, notes: map[string]string{}, allPkgs: len(pkgs)}
	changedPkgs := map[string][]string{}
	var unowned []string
	for _, f := range changed {
		if f == "go.mod" || f == "go.sum" {
			sel.wholeMod = true
			sel.mapped[f] = ""
			sel.notes[f] = "module file: every package"
			continue
		}
		dir := filepath.Dir(filepath.Join(repo, f))
		if strings.HasSuffix(f, ".go") && !atIgnoredByGo(f) {
			if _, listed := byDir[dir]; !listed {
				if st, err := os.Stat(dir); err == nil && st.IsDir() {
					sel.mapped[f] = ""
					sel.notes[f] = "build constraints exclude its package natively (wasm: ./.github/wasm_test.sh)"
					continue
				}
			}
		}
		owner := ""
		for d := dir; strings.HasPrefix(d, repo) && d != repo; d = filepath.Dir(d) {
			if p, ok := byDir[d]; ok {
				owner = p.ImportPath
				break
			}
		}
		sel.mapped[f] = owner
		if owner == "" {
			sel.notes[f] = "no Go package owns it"
			unowned = append(unowned, f)
			continue
		}
		changedPkgs[owner] = append(changedPkgs[owner], f)
	}

	for _, p := range pkgs {
		if sel.wholeMod {
			sel.picked = append(sel.picked, atPick{p, "go.mod/go.sum changed"})
			continue
		}
		if files, ok := changedPkgs[p.ImportPath]; ok {
			sel.picked = append(sel.picked, atPick{p, "changed: " + strings.Join(files, " ")})
			continue
		}
		reach := map[string]bool{}
		for _, d := range p.Deps {
			reach[d] = true
		}
		for _, ti := range append(append([]string{}, p.TestImports...), p.XTestImports...) {
			reach[ti] = true
			if tp, ok := byPath[ti]; ok {
				for _, d := range tp.Deps {
					reach[d] = true
				}
			}
		}
		var via []string
		for c := range changedPkgs {
			if reach[c] {
				via = append(via, atShort(c))
			}
		}
		if len(via) > 0 {
			sort.Strings(via)
			sel.picked = append(sel.picked, atPick{p, "imports " + strings.Join(via, " ")})
			continue
		}
		if len(unowned) > 0 {
			var names []string
			lits := atStringLiterals(p)
			for _, f := range unowned {
				base := filepath.Base(f)
				for _, l := range lits {
					if l == base || strings.HasSuffix(l, "/"+base) {
						names = append(names, base)
						sel.notes[f] += "; named by " + atShort(p.ImportPath)
						break
					}
				}
			}
			if len(names) > 0 {
				sel.picked = append(sel.picked, atPick{p, "names " + strings.Join(names, " ") + " in a string literal"})
			}
		}
	}
	sort.Slice(sel.picked, func(i, j int) bool { return sel.picked[i].pkg.ImportPath < sel.picked[j].pkg.ImportPath })
	return sel
}

func atStringLiterals(p *atPackage) []string {
	var lits []string
	for _, name := range append(append(append([]string{}, p.GoFiles...), p.TestGoFiles...), p.XTestGoFiles...) {
		src, err := os.ReadFile(filepath.Join(p.Dir, name))
		if err != nil {
			continue
		}
		var sc scanner.Scanner
		fset := token.NewFileSet()
		sc.Init(fset.AddFile(name, -1, len(src)), src, nil, 0)
		for {
			_, tok, lit := sc.Scan()
			if tok == token.EOF {
				break
			}
			if tok == token.STRING {
				if u, err := strconv.Unquote(lit); err == nil {
					lits = append(lits, u)
				}
			}
		}
	}
	return lits
}

func atShort(importPath string) string {
	return strings.TrimPrefix(strings.TrimPrefix(importPath, root.Module), "/")
}

func affectedTests(args []string) int {
	dry := false
	var goArgs []string
	for _, a := range args {
		if a == "--dry" {
			dry = true
			continue
		}
		goArgs = append(goArgs, a)
	}
	repo, dirs, rc := atAffected()
	if rc != 0 || len(dirs) == 0 {
		return rc
	}
	cmd := append(append([]string{"go", "test"}, dirs...), goArgs...)
	fmt.Println(strings.Join(cmd, " "))
	if dry {
		return 0
	}
	c := exec.CommandContext(context.Background(), cmd[0], cmd[1:]...) //nolint:gosec // go test with the caller's own arguments
	c.Dir = repo
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		return atRefuse(err.Error())
	}
	return 0
}

func atAffected() (string, []string, int) {
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return "", nil, root.RefusalCode
	}
	repo, err = filepath.EvalSymlinks(repo)
	if err != nil {
		return "", nil, atRefuse(err.Error())
	}

	base := os.Getenv("BASE")
	baseFrom := "BASE"
	if base == "" {
		mb, err := atGit(repo, "merge-base", "dev", "HEAD")
		if err != nil || len(mb) != 1 {
			//go-only: a new tool with no original; it cannot pick a base.
			return "", nil, atRefuse(fmt.Sprintf("no merge-base with dev; set BASE=<ref> (%v)", err))
		}
		base, baseFrom = mb[0], "merge-base dev HEAD"
	}
	if _, err := atGit(repo, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		//go-only: a new tool with no original; BASE must name a commit.
		return "", nil, atRefuse(fmt.Sprintf("BASE %q is not a commit", base))
	}

	seen := map[string]bool{}
	var changed []string
	for _, q := range [][]string{
		{"diff", "--name-only", base + "...HEAD"},
		{"diff", "--name-only", "HEAD"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		lines, err := atGit(repo, q...)
		if err != nil {
			return "", nil, atRefuse(err.Error())
		}
		for _, l := range lines {
			if !seen[l] {
				seen[l] = true
				changed = append(changed, l)
			}
		}
	}
	sort.Strings(changed)

	pkgs, err := atListPackages(repo)
	if err != nil {
		return "", nil, atRefuse(err.Error())
	}
	sel := atSelect(repo, pkgs, changed)

	fmt.Printf("affected-tests: base %s (%s), %d changed files\n", base, baseFrom, len(changed))
	for _, f := range changed {
		if sel.mapped[f] != "" {
			fmt.Printf("  %-60s -> %s\n", f, atShort(sel.mapped[f]))
		} else {
			fmt.Printf("  %-60s -> none: %s\n", f, sel.notes[f])
		}
	}
	if len(sel.picked) == 0 {
		cause := "no changes since base"
		if len(changed) > 0 {
			cause = "no changed file belongs to a Go package"
		}
		fmt.Printf("selected 0 of %d packages: %s; nothing to test\n", sel.allPkgs, cause)
		return repo, nil, 0
	}
	fmt.Printf("selected %d of %d packages:\n", len(sel.picked), sel.allPkgs)
	var dirs []string
	for _, p := range sel.picked {
		fmt.Printf("  %-40s %s\n", atShort(p.pkg.ImportPath), p.reason)
		rel, err := filepath.Rel(repo, p.pkg.Dir)
		if err != nil {
			return "", nil, atRefuse(err.Error())
		}
		dirs = append(dirs, "./"+filepath.ToSlash(rel))
	}
	return repo, dirs, 0
}
