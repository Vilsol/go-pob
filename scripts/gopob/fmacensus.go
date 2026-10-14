package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

const fmaModule = "github.com/Vilsol/go-pob/"

var fmaPackages = []string{"calculator", "moddb", "mod", "data", "pob", "builds", "utils"}

type fmaTarget struct {
	name     string
	env      []string
	mnemonic *regexp.Regexp
}

var fmaTargets = []fmaTarget{
	{"arm64", []string{"GOOS=linux", "GOARCH=arm64"}, regexp.MustCompile(`\t(FN?M(?:ADD|SUB)[DS])\t`)},
	{"amd64v3", []string{"GOOS=linux", "GOARCH=amd64", "GOAMD64=v3"}, regexp.MustCompile(`\t(VFN?M(?:ADD|SUB)\d+S[DS])\t`)},
}

var (
	fmaPos  = regexp.MustCompile(`\((` + regexp.QuoteMeta(fmaModule) + `[^)]+\.go):(\d+)\)`)
	fmaText = regexp.MustCompile(`\sSTEXT\s`)
)

var fmaAllow = map[string]string{
	"calculator/calc_defence.go" + "\t" + "enemyDamage = enemyDamage + (actor.Enemy.ModDB.Sum(mod.TypeBase, enemyCfg, string(damageType+\"Min\"))+actor.Enemy.ModDB.Sum(mod.TypeBase, enemyCfg, string(damageType+\"Max\")))/2": fmaPow2,
	"calculator/calc_defence.go" + "\t" + "ExtraAvoidChance = ExtraAvoidChance + actor.Output[\"AvoidProjectilesChance\"]/2":                                                                                                    fmaPow2,
	"calculator/calc_defence.go" + "\t" + "modDB.Sum(mod.TypeBase, nil, \"DamageTakenFromAttacks\", string(damageType+\"DamageTakenFromAttacks\"))/2 +":                                                                         fmaPow2,
	"calculator/calc_defence.go" + "\t" + "modDB.Sum(mod.TypeBase, nil, \"DamageTakenFromSpells\", string(damageType+\"DamageTakenFromSpells\"))/2 +":                                                                           fmaPow2,
	"calculator/calc_defence.go" + "\t" + "modDB.Sum(mod.TypeBase, nil, string(damageType+\"DamageTakenFromProjectileAttacks\"))/4 +":                                                                                           fmaPow2,
	"calculator/calc_defence.go" + "\t" + "spellSuppressMult = utils.Ternary(actor.Output[\"EffectiveSpellSuppressionChance\"] == 100, 1-actor.Output[\"SpellSuppressionEffect\"]/100/2, float64(1))":                           fmaPow2,
	"calculator/calc_defence.go" + "\t" + "takenFlat = takenFlat +":                                                                                                                                                             fmaPow2,
	"calculator/calc_offence.go" + "\t" + "bleedRollAverage = float64((bleedStacks - (maxStacks-1)/2) / (bleedStacks + 1) * 100)":                                                                                               fmaPow2,
	"calculator/calc_offence.go" + "\t" + "igniteRollAverage = float64((igniteStacks - (maxStacks-1)/2) / (igniteStacks + 1) * 100)":                                                                                            fmaPow2,
	"calculator/calc_offence.go" + "\t" + "modValue = (float64((100+raw*(repeats-2)/2)*(repeats-1))+":                                                                                                                           fmaPow2,
	"calculator/calc_perform.go" + "\t" + "avoid += modDB.Sum(mod.TypeBase, nil, \"SpellSuppressionChance\") / 2":                                                                                                               fmaPow2,
}

const (
	fmaPow2 = "a power-of-two divide compiles to an exact multiply by 2^-k, so the fused add rounds once either way"
)

type fmaSite struct {
	target, file, mnemonic, code string
	line                         int
}

func (s fmaSite) key() string { return s.file + "\t" + s.code }

//go-only: fma-census is Go-native; it has no original.

func fmaRefuse(msg string) int {
	fmt.Fprintln(os.Stderr, "fma-census: "+msg)
	return root.RefusalCode
}

func fmaCensus(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: fma-census takes no arguments, got %q\n", args[0])
		return 64
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}

	var sites []fmaSite
	for _, tg := range fmaTargets {
		got, rc := fmaCompile(repo, tg)
		if rc != 0 {
			return rc
		}
		sites = append(sites, got...)
	}

	seen := map[string]bool{}
	var bad []fmaSite
	for _, s := range sites {
		seen[s.key()] = true
		if _, ok := fmaAllow[s.key()]; !ok {
			bad = append(bad, s)
		}
	}
	sort.Slice(bad, func(i, j int) bool {
		a, b := bad[i], bad[j]
		if a.file != b.file {
			return a.file < b.file
		}
		if a.line != b.line {
			return a.line < b.line
		}
		return a.target < b.target
	})
	for _, s := range bad {
		fmt.Printf("FUSED  %-8s %s:%d  %-12s %s\n", s.target, s.file, s.line, s.mnemonic, s.code)
	}
	var stale []string
	for k := range fmaAllow {
		if !seen[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(stale)
	for _, k := range stale {
		fmt.Printf("STALE  allow-list entry no longer fuses: %s\n", strings.ReplaceAll(k, "\t", "  "))
	}

	fmt.Printf("%d fused instruction(s) across %s; %d allow-listed, %d not, %d stale allow-list entr(ies)\n",
		len(sites), fmaTargetNames(), len(sites)-len(bad), len(bad), len(stale))
	if len(bad) > 0 || len(stale) > 0 {
		fmt.Println("wrap the product in an explicit conversion, float64(x*y), so it rounds before the add as PoB's Lua does")
		return 1
	}
	return 0
}

func fmaTargetNames() string {
	names := make([]string, len(fmaTargets))
	for i, tg := range fmaTargets {
		names[i] = tg.name
	}
	return strings.Join(names, ", ")
}

func fmaCompile(repo string, tg fmaTarget) ([]fmaSite, int) {
	argv := make([]string, 0, 1+2*len(fmaPackages))
	argv = append(argv, "build", "-trimpath")
	pkgs := make([]string, 0, len(fmaPackages))
	for _, p := range fmaPackages {
		argv = append(argv, "-gcflags="+fmaModule+p+"/...=-S")
		pkgs = append(pkgs, "./"+p+"/...")
	}
	argv = append(argv, pkgs...)
	cmd := exec.Command("go", argv...) //nolint:gosec,noctx // a fixed go build
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), tg.env...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		//go-only: fma-census is Go-native; it has no original.
		return nil, fmaRefuse(fmt.Sprintf("%s: go build failed: %v\n%s", tg.name, err, out.String()))
	}

	var sites []fmaSite
	texts := 0
	sources := map[string][]string{}
	sc := bufio.NewScanner(&out)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		ln := sc.Text()
		if fmaText.MatchString(ln) {
			texts++
		}
		m := tg.mnemonic.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		p := fmaPos.FindStringSubmatch(ln)
		if p == nil {
			//go-only: fma-census is Go-native; it has no original.
			return nil, fmaRefuse(fmt.Sprintf("%s: a fused instruction with no module source position: %s", tg.name, ln))
		}
		file := strings.TrimPrefix(p[1], fmaModule)
		n, _ := strconv.Atoi(p[2])
		code, err := fmaSourceLine(repo, file, n, sources)
		if err != nil {
			//go-only: fma-census is Go-native; it has no original.
			return nil, fmaRefuse(fmt.Sprintf("%s: %v", tg.name, err))
		}
		sites = append(sites, fmaSite{target: tg.name, file: file, line: n, mnemonic: m[1], code: code})
	}
	if texts == 0 {
		//go-only: fma-census is Go-native; it has no original.
		return nil, fmaRefuse(fmt.Sprintf("%s: go build printed no function bodies, so -S did not reach the compiler", tg.name))
	}
	return sites, 0
}

func fmaSourceLine(repo, file string, n int, cache map[string][]string) (string, error) {
	lines, ok := cache[file]
	if !ok {
		b, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(file))) //nolint:gosec // a path the compiler printed
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", file, err)
		}
		lines = strings.Split(string(b), "\n")
		cache[file] = lines
	}
	if n < 1 || n > len(lines) {
		return "", fmt.Errorf("%s has no line %d", file, n)
	}
	return strings.TrimSpace(lines[n-1]), nil
}
