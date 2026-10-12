package datafields

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const (
	ModulePath    = "github.com/Vilsol/go-pob"
	DataModule    = "github.com/Vilsol/go-pob-data"
	RawPkg        = DataModule + "/raw"
	loaderPkg     = DataModule + "/loader"
	generatedFile = "raw_gen.go"
	ourRawPkg     = ModulePath + "/data/raw"
	selfPkg       = ModulePath + "/scripts/gopob/internal/datafields"
	generatorPkg  = ModulePath + "/scripts/gamedatadump"
)

var DataPackages = []string{RawPkg, DataModule + "/poe", loaderPkg, DataModule + "/utils", DataModule + "/fields"}

var BuildConfigs = []BuildConfig{
	{Name: "native", Patterns: []string{"./..."}},
	{Name: "js/wasm", Env: []string{"GOOS=js", "GOARCH=wasm"}, Closure: "./wasm/..."},
}

type BuildConfig struct {
	Name     string
	Env      []string
	Patterns []string
	Closure  string
}

type FieldKey struct {
	Type  string
	Field string
}

type Site struct {
	Pos  string
	Test bool
}

type Whole struct {
	Why  string
	Site Site
}

type Asset struct {
	Path    string
	RowType string
	Site    Site
}

type Result struct {
	Structs map[string]*StructInfo
	Reads   map[FieldKey][]Site
	Whole   map[string][]Whole
	Assets  map[string]Asset
	Allowed map[string][]Site

	DataModuleVersion string
	CacheNamespace    string
	DefaultVersion    string
}

type StructInfo struct {
	Name   string
	Fields []FieldInfo
}

type FieldInfo struct {
	Name   string
	Nested string
}

type RefusalError struct{ Reasons []string }

func (e *RefusalError) Error() string {
	return "data-fields REFUSES to emit a list:\n  " + strings.Join(e.Reasons, "\n  ")
}

type allowEntry struct {
	fn     string
	callee string
	why    string
}

var allowlist = []allowEntry{
	{
		fn:     loaderPkg + ".LoadRaw",
		callee: "reflect.ValueOf",
		why:    "reflect.ValueOf(out[0]).Type().Elem() only reads the element TYPE to allocate rows",
	},
	{
		fn:     loaderPkg + ".LoadRaw",
		callee: loaderPkg + ".VariantURL",
		why:    "VariantURL hands the zero row only to LayoutHash, which reads its TYPE to name the asset",
	},
	{
		fn:     loaderPkg + ".RawURL",
		callee: loaderPkg + ".TupleVariant",
		why:    "TupleVariant hands the zero row only to LayoutHash, which reads its TYPE",
	},
	{
		fn:     loaderPkg + ".TranslationURL",
		callee: loaderPkg + ".TupleVariant",
		why:    "TupleVariant hands the zero row only to LayoutHash, which reads its TYPE",
	},
	{
		fn:     loaderPkg + ".LoadTranslation",
		callee: loaderPkg + ".VariantURL",
		why:    "VariantURL hands the zero row only to LayoutHash, which reads its TYPE to name the asset",
	},
}

func Analyze(dir string) (*Result, error) {
	return analyze(dir, allowlist)
}

func analyze(dir string, allow []allowEntry) (*Result, error) {
	res := &Result{
		Structs: map[string]*StructInfo{},
		Reads:   map[FieldKey][]Site{},
		Whole:   map[string][]Whole{},
		Assets:  map[string]Asset{},
		Allowed: map[string][]Site{},
	}
	var refusals []string
	allowHits := map[int]bool{}

	for _, bc := range BuildConfigs {
		r, err := analyzeConfig(dir, bc, res, allow, allowHits)
		if err != nil {
			return nil, err
		}
		refusals = append(refusals, r...)
	}

	for i, a := range allow {
		if !allowHits[i] {
			refusals = append(refusals, fmt.Sprintf("allowlist entry %s -> %s matched no site; delete it", a.fn, a.callee))
		}
	}
	if len(res.Assets) == 0 {
		refusals = append(refusals, "no asset loads found; the loader call sites moved")
	}
	for k := range res.Reads {
		if _, ok := res.Structs[k.Type]; !ok {
			refusals = append(refusals, "read of unknown raw type "+k.Type)
		}
	}

	if len(refusals) > 0 {
		sort.Strings(refusals)
		return nil, &RefusalError{Reasons: dedupe(refusals)}
	}
	return res, nil
}

func analyzeConfig(dir string, bc BuildConfig, res *Result, allow []allowEntry, allowHits map[int]bool) ([]string, error) {
	env := append(envWithout("GOOS", "GOARCH"), bc.Env...)
	patterns := bc.Patterns
	if bc.Closure != "" {
		closure, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps, Dir: dir, Env: env}, bc.Closure)
		if err != nil {
			return nil, fmt.Errorf("%s: load closure: %w", bc.Name, err)
		}
		packages.Visit(closure, nil, func(p *packages.Package) {
			if inModule(p.PkgPath) {
				patterns = append(patterns, p.PkgPath)
			}
		})
		if len(patterns) == 0 {
			return []string{bc.Name + ": the closure of " + bc.Closure + " holds no go-pob package"}, nil
		}
	}
	cfg := &packages.Config{
		Mode:  packages.LoadSyntax | packages.NeedModule,
		Dir:   dir,
		Tests: true,
		Env:   env,
	}
	pkgs, err := packages.Load(cfg, append(patterns, DataPackages...)...)
	if err != nil {
		return nil, fmt.Errorf("%s: load: %w", bc.Name, err)
	}

	var refusals []string
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		if !inModule(p.PkgPath) && !isDataPackage(strings.TrimSuffix(p.PkgPath, "_test")) {
			return
		}
		for _, e := range p.Errors {
			refusals = append(refusals, fmt.Sprintf("%s: package %s does not load: %s", bc.Name, p.ID, e))
		}
		if !inModule(p.PkgPath) {
			return
		}
		for _, imp := range p.Imports {
			if strings.HasPrefix(imp.PkgPath, DataModule+"/") && !isDataPackage(imp.PkgPath) {
				refusals = append(refusals, fmt.Sprintf("%s: %s imports %s, which is not analysed; add it to DataPackages", bc.Name, p.ID, imp.PkgPath))
			}
		}
	})
	if len(refusals) > 0 {
		return refusals, nil
	}

	for _, p := range pkgs {
		if p.PkgPath == RawPkg && p.Types != nil {
			collectStructs(p.Types, res)
			if p.Module != nil {
				res.DataModuleVersion = p.Module.Version
			}
		}
		if p.PkgPath == ourRawPkg && p.Types != nil {
			res.CacheNamespace = constOf(p.Types, "DataModuleVersion")
			res.DefaultVersion = constOf(p.Types, "DefaultVersion")
		}
	}
	if res.DataModuleVersion == "" || res.CacheNamespace != res.DataModuleVersion || res.DefaultVersion == "" {
		refusals = append(refusals, fmt.Sprintf("%s: go.mod requires %s %q but %s.DataModuleVersion is %q (default game version %q); the cache namespace would not match the assets",
			bc.Name, DataModule, res.DataModuleVersion, ourRawPkg, res.CacheNamespace, res.DefaultVersion))
	}
	if len(res.Structs) == 0 {
		return []string{bc.Name + ": the raw package declared no struct types"}, nil
	}

	prog, _ := ssautil.Packages(pkgs, ssa.InstantiateGenerics)
	prog.Build()

	a := &analyzer{prog: prog, fset: prog.Fset, res: res, cfg: bc.Name, allow: allow, allowHits: allowHits, root: absDir(dir)}
	all := ssautil.AllFunctions(prog)
	reach := reachable(prog, all, func(fn *ssa.Function) bool {
		path := strings.TrimSuffix(pkgPathOf(fn), "_test")
		return fn.Blocks != nil && inModule(path) && a.owned(fn)
	}, func(mi *ssa.MakeInterface, fn *ssa.Function) bool {
		_, ok := a.allowed(mi, fn)
		return ok || a.site(mi.Pos(), fn).Test
	})
	for fn := range all {
		if fn.Blocks == nil || !a.owned(fn) {
			continue
		}
		if isDataPackage(strings.TrimSuffix(pkgPathOf(fn), "_test")) && !reach[fn] {
			continue
		}
		a.scan(fn)
	}
	return append(refusals, a.refusals...), nil
}

type analyzer struct {
	prog      *ssa.Program
	fset      *token.FileSet
	res       *Result
	cfg       string
	refusals  []string
	allow     []allowEntry
	allowHits map[int]bool
	root      string
}

func (a *analyzer) owned(fn *ssa.Function) bool {
	file := a.fset.Position(fn.Pos()).Filename
	path := pkgPathOf(fn)
	if path == "" {
		return true
	}
	path = strings.TrimSuffix(path, "_test")
	if isDataPackage(path) {
		return filepath.Base(file) != generatedFile && !strings.HasSuffix(file, "_test.go")
	}
	return inModule(path) && path != selfPkg && path != generatorPkg
}

func inModule(path string) bool {
	return path == ModulePath || strings.HasPrefix(path, ModulePath+"/")
}

func (a *analyzer) site(pos token.Pos, fn *ssa.Function) Site {
	if !pos.IsValid() {
		pos = fn.Pos()
	}
	p := a.fset.Position(pos)
	file := p.Filename
	if rel, err := filepath.Rel(a.root, file); err == nil && !strings.HasPrefix(rel, "..") {
		file = rel
	} else if i := strings.Index(file, "go-pob-data@"); i >= 0 {
		file = file[i:]
	}
	return Site{Pos: fmt.Sprintf("%s:%d", file, p.Line), Test: strings.HasSuffix(p.Filename, "_test.go")}
}

func (a *analyzer) scan(fn *ssa.Function) {
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			switch in := instr.(type) {
			case *ssa.FieldAddr:
				if !writeOnly(in) {
					a.field(in.X.Type(), in.Field, in.Pos(), fn)
				}
			case *ssa.Field:
				a.field(in.X.Type(), in.Field, in.Pos(), fn)
			case *ssa.MakeInterface:
				a.makeInterface(in, fn)
			case *ssa.Convert:
				if isUnsafePointer(in.Type()) || isUnsafePointer(in.X.Type()) {
					a.wholeOf(in.X.Type(), "unsafe.Pointer conversion", in.Pos(), fn)
				}
			case *ssa.BinOp:
				if in.Op == token.EQL || in.Op == token.NEQ {
					a.wholeInline(in.X.Type(), "== / != compares every field", in.Pos(), fn)
				}
			case *ssa.Lookup:
				if m, ok := in.X.Type().Underlying().(*types.Map); ok {
					a.wholeInline(m.Key(), "used as a map key (hashed)", in.Pos(), fn)
				}
			case *ssa.MapUpdate:
				if m, ok := in.Map.Type().Underlying().(*types.Map); ok {
					a.wholeInline(m.Key(), "used as a map key (hashed)", in.Pos(), fn)
				}
			}
			if call, ok := instr.(ssa.CallInstruction); ok {
				a.call(call, fn)
			}
		}
	}
}

func writeOnly(fa *ssa.FieldAddr) bool {
	refs := fa.Referrers()
	if refs == nil || len(*refs) == 0 {
		return false
	}
	for _, r := range *refs {
		st, ok := r.(*ssa.Store)
		if !ok || st.Addr != fa || st.Val == fa {
			return false
		}
	}
	return true
}

func (a *analyzer) field(t types.Type, idx int, pos token.Pos, fn *ssa.Function) {
	named := namedOf(deref(t))
	if named == nil || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != RawPkg {
		return
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return
	}
	k := FieldKey{Type: named.Obj().Name(), Field: st.Field(idx).Name()}
	a.res.Reads[k] = appendSite(a.res.Reads[k], a.site(pos, fn))
}

func (a *analyzer) makeInterface(mi *ssa.MakeInterface, fn *ssa.Function) {
	raws := rawTypesIn(mi.X.Type(), true)
	if len(raws) == 0 {
		return
	}
	if i, ok := a.allowed(mi, fn); ok {
		a.allowHits[i] = true
		key := a.allow[i].fn + " -> " + a.allow[i].callee
		a.res.Allowed[key] = appendSite(a.res.Allowed[key], a.site(mi.Pos(), fn))
		return
	}
	pos := mi.Pos()
	if refs := mi.Referrers(); !pos.IsValid() && refs != nil {
		for _, r := range *refs {
			if r.Pos().IsValid() {
				pos = r.Pos()
				break
			}
		}
	}
	why := fmt.Sprintf("%s converted to %s in %s (reflection or an unfollowed dynamic call may read any field)", mi.X.Type(), mi.Type(), fnName(fn))
	for _, r := range raws {
		a.addWhole(r, why, pos, fn)
	}
}

func (a *analyzer) allowed(mi *ssa.MakeInterface, fn *ssa.Function) (int, bool) {
	name := fnName(fn)
	refs := mi.Referrers()
	if refs == nil || len(*refs) == 0 {
		return 0, false
	}
	for i, e := range a.allow {
		if name != e.fn {
			continue
		}
		all := true
		for _, r := range *refs {
			call, ok := r.(ssa.CallInstruction)
			if !ok || call.Common().StaticCallee() == nil || call.Common().StaticCallee().String() != e.callee {
				all = false
				break
			}
		}
		if all {
			return i, true
		}
	}
	return 0, false
}

func (a *analyzer) call(call ssa.CallInstruction, fn *ssa.Function) {
	common := call.Common()
	callee := common.StaticCallee()
	if callee == nil {
		return
	}
	origin := callee
	if callee.Origin() != nil {
		origin = callee.Origin()
	}

	if pkgPathOf(origin) == RawPkg && filepath.Base(a.fset.Position(origin.Pos()).Filename) == generatedFile {
		switch origin.Name() {
		case "DecodeMsg", "UnmarshalMsg":
		default:
			if sig := origin.Signature; sig.Recv() != nil {
				a.wholeOf(sig.Recv().Type(), "msgp "+origin.Name()+" (encodes every field)", call.Pos(), fn)
			} else {
				a.refuse(call.Pos(), fn, "call to generated function "+origin.Name())
			}
		}
	}

	if callee.Blocks == nil && len(callee.TypeArgs()) > 0 {
		for _, ta := range callee.TypeArgs() {
			if len(rawTypesIn(ta, true)) == 0 {
				continue
			}
			if safe, compares := safeGeneric(origin); safe {
				if compares {
					a.wholeInline(ta, "compared by "+origin.String(), call.Pos(), fn)
				}
				continue
			}
			a.wholeOf(ta, "instantiates external generic "+origin.String()+" whose body is not analysed", call.Pos(), fn)
		}
	}

	switch origin.String() {
	case loaderPkg + ".InitHelper", loaderPkg + ".LoadRaw":
		a.asset(common, callee, fn, "raw/", 2, -1)
	case loaderPkg + ".LoadTranslation":
		a.asset(common, callee, fn, "stat_translations/", 3, 2)
	}
}

func safeGeneric(fn *ssa.Function) (bool, bool) {
	switch pkgPathOf(fn) {
	case "slices", "maps", "sync", "sync/atomic", "iter", "sort", "cmp":
		switch fn.Name() {
		case "Contains", "Index", "Equal", "Compact":
			return true, true
		}
		return true, false
	case "reflect":
		if fn.Name() == "TypeFor" {
			return true, false
		}
	}
	return false, false
}

func (a *analyzer) asset(common *ssa.CallCommon, callee *ssa.Function, fn *ssa.Function, prefix string, nameArg, langArg int) {
	if pkgPathOf(fn) == loaderPkg {
		return
	}
	name, ok := constString(common.Args, nameArg)
	if !ok {
		a.refuse(common.Pos(), fn, "asset name is not a constant in a call to "+callee.String())
		return
	}
	path := prefix + name
	rowType := "TranslationFile"
	if langArg >= 0 {
		lang, ok := constString(common.Args, langArg)
		if !ok {
			a.refuse(common.Pos(), fn, "translation language is not a constant")
			return
		}
		path = prefix + lang + "/" + name
	} else {
		targs := callee.TypeArgs()
		if len(targs) != 1 {
			a.refuse(common.Pos(), fn, "unexpected type arguments to "+callee.String())
			return
		}
		rowType = rowTypeOf(targs[0])
		if rowType == "" {
			a.refuse(common.Pos(), fn, fmt.Sprintf("cannot map %s to a raw row type", targs[0]))
			return
		}
	}
	s := a.site(common.Pos(), fn)
	if prev, ok := a.res.Assets[path]; ok && prev.RowType != rowType {
		a.refuse(common.Pos(), fn, fmt.Sprintf("asset %s loaded as both %s and %s", path, prev.RowType, rowType))
		return
	}
	if prev, ok := a.res.Assets[path]; !ok || (prev.Site.Test && !s.Test) {
		a.res.Assets[path] = Asset{Path: path, RowType: rowType, Site: s}
	}
}

func rowTypeOf(t types.Type) string {
	named := namedOf(deref(t))
	if named == nil {
		return ""
	}
	if named.Obj().Pkg().Path() == RawPkg {
		return named.Obj().Name()
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return ""
	}
	found := ""
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Embedded() {
			continue
		}
		if n := namedOf(deref(f.Type())); n != nil && n.Obj().Pkg().Path() == RawPkg {
			if found != "" {
				return ""
			}
			found = n.Obj().Name()
		}
	}
	return found
}

func constString(args []ssa.Value, i int) (string, bool) {
	if i >= len(args) {
		return "", false
	}
	c, ok := args[i].(*ssa.Const)
	if !ok || c.Value == nil || c.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(c.Value), true
}

func (a *analyzer) wholeOf(t types.Type, why string, pos token.Pos, fn *ssa.Function) {
	for _, r := range rawTypesIn(t, true) {
		a.addWhole(r, why, pos, fn)
	}
}

func (a *analyzer) wholeInline(t types.Type, why string, pos token.Pos, fn *ssa.Function) {
	for _, r := range rawTypesIn(t, false) {
		a.addWhole(r, why, pos, fn)
	}
}

func (a *analyzer) addWhole(typ, why string, pos token.Pos, fn *ssa.Function) {
	s := a.site(pos, fn)
	for _, w := range a.res.Whole[typ] {
		if w.Site == s && w.Why == why {
			return
		}
	}
	a.res.Whole[typ] = append(a.res.Whole[typ], Whole{Why: why, Site: s})
}

func (a *analyzer) refuse(pos token.Pos, fn *ssa.Function, msg string) {
	a.refusals = append(a.refusals, fmt.Sprintf("%s: %s: %s", a.cfg, a.site(pos, fn).Pos, msg))
}

func rawTypesIn(t types.Type, throughPointers bool) []string {
	seen := map[types.Type]bool{}
	out := map[string]bool{}
	var walk func(types.Type)
	walk = func(t types.Type) {
		t = types.Unalias(t)
		if seen[t] {
			return
		}
		seen[t] = true
		if n, ok := t.(*types.Named); ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == RawPkg {
			if _, ok := n.Underlying().(*types.Struct); ok {
				out[n.Obj().Name()] = true
			}
		}
		switch u := t.Underlying().(type) {
		case *types.Pointer:
			if throughPointers {
				walk(u.Elem())
			}
		case *types.Slice:
			if throughPointers {
				walk(u.Elem())
			}
		case *types.Array:
			walk(u.Elem())
		case *types.Map:
			if throughPointers {
				walk(u.Key())
				walk(u.Elem())
			}
		case *types.Chan:
			if throughPointers {
				walk(u.Elem())
			}
		case *types.Struct:
			for i := 0; i < u.NumFields(); i++ {
				walk(u.Field(i).Type())
			}
		}
	}
	walk(t)
	names := make([]string, 0, len(out))
	for n := range out {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func collectStructs(pkg *types.Package, res *Result) {
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		st, ok := tn.Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}
		info := &StructInfo{Name: name}
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			info.Fields = append(info.Fields, FieldInfo{Name: f.Name(), Nested: nestedStruct(f.Type())})
		}
		res.Structs[name] = info
	}
}

func constOf(pkg *types.Package, name string) string {
	c, ok := pkg.Scope().Lookup(name).(*types.Const)
	if !ok || c.Val().Kind() != constant.String {
		return ""
	}
	return constant.StringVal(c.Val())
}

func absDir(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}

func envWithout(keys ...string) []string {
	var out []string
	for _, kv := range os.Environ() {
		drop := false
		for _, k := range keys {
			if strings.HasPrefix(kv, k+"=") {
				drop = true
			}
		}
		if !drop {
			out = append(out, kv)
		}
	}
	return out
}

func nestedStruct(t types.Type) string {
	for {
		t = types.Unalias(t)
		if n, ok := t.(*types.Named); ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == RawPkg {
			if _, ok := n.Underlying().(*types.Struct); ok {
				return n.Obj().Name()
			}
		}
		switch u := t.Underlying().(type) {
		case *types.Pointer:
			t = u.Elem()
		case *types.Slice:
			t = u.Elem()
		case *types.Array:
			t = u.Elem()
		default:
			return ""
		}
	}
}

func isDataPackage(path string) bool {
	for _, p := range DataPackages {
		if p == path {
			return true
		}
	}
	return false
}

func pkgPathOf(fn *ssa.Function) string {
	if fn.Origin() != nil {
		fn = fn.Origin()
	}
	if fn.Pkg != nil {
		return fn.Pkg.Pkg.Path()
	}
	if obj := fn.Object(); obj != nil && obj.Pkg() != nil {
		return obj.Pkg().Path()
	}
	return ""
}

func fnName(fn *ssa.Function) string {
	for fn.Parent() != nil {
		fn = fn.Parent()
	}
	if fn.Origin() != nil {
		fn = fn.Origin()
	}
	return fn.String()
}

func deref(t types.Type) types.Type {
	if p, ok := types.Unalias(t).Underlying().(*types.Pointer); ok {
		return p.Elem()
	}
	return t
}

func namedOf(t types.Type) *types.Named {
	n, _ := types.Unalias(t).(*types.Named)
	return n
}

func isUnsafePointer(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.UnsafePointer
}

func appendSite(sites []Site, s Site) []Site {
	for _, x := range sites {
		if x == s {
			return sites
		}
	}
	return append(sites, s)
}

func dedupe(in []string) []string {
	out := in[:0]
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
