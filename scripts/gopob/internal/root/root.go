package root

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Module = "github.com/Vilsol/go-pob"

const (
	Refusal = "REFUSING: no go.mod declaring module " + Module +
		" at or above this tool; set GOPOB_ROOT to the checkout"
	RefusalEnv  = "REFUSING: GOPOB_ROOT does not contain a go.mod declaring module " + Module
	RefusalCode = 3
)

type refusal string

func (r refusal) Error() string { return string(r) }

const ErrNotFound = refusal(Refusal)

const ErrBadEnv = refusal(RefusalEnv)

func declaresModule(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "module "+Module {
			return true
		}
	}
	return false
}

func From(anchor string) (string, error) {
	if env := os.Getenv("GOPOB_ROOT"); env != "" {
		abs, err := filepath.Abs(env)
		if err != nil || !declaresModule(abs) {
			return "", ErrBadEnv
		}
		return abs, nil
	}
	dir, err := filepath.Abs(anchor)
	if err != nil {
		return "", ErrNotFound
	}
	for {
		if declaresModule(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

func Resolve() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", ErrNotFound
	}
	return From(cwd)
}

func MustResolve() string {
	r, err := Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(RefusalCode)
	}
	return r
}
