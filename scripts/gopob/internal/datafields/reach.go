package datafields

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

func reachable(prog *ssa.Program, all map[*ssa.Function]bool, root func(*ssa.Function) bool, typeOnly func(*ssa.MakeInterface, *ssa.Function) bool) map[*ssa.Function]bool {
	methodsByID := map[string][]*ssa.Function{}
	for f := range all {
		if f.Signature.Recv() != nil && f.Object() != nil {
			id := f.Object().(*types.Func).Id()
			methodsByID[id] = append(methodsByID[id], f)
		}
	}

	seen := map[*ssa.Function]bool{}
	var queue []*ssa.Function
	add := func(f *ssa.Function) {
		if f != nil && !seen[f] {
			seen[f] = true
			queue = append(queue, f)
		}
	}
	typeSeen := map[types.Type]bool{}
	var addMethods func(types.Type)
	addMethods = func(t types.Type) {
		t = types.Unalias(t)
		if typeSeen[t] {
			return
		}
		typeSeen[t] = true
		if _, ok := t.Underlying().(*types.Interface); !ok {
			for _, mt := range []types.Type{t, types.NewPointer(t)} {
				ms := prog.MethodSets.MethodSet(mt)
				for i := 0; i < ms.Len(); i++ {
					if !ms.At(i).Obj().Exported() {
						continue
					}
					add(prog.MethodValue(ms.At(i)))
				}
			}
		}
		switch u := t.Underlying().(type) {
		case *types.Pointer:
			addMethods(u.Elem())
		case *types.Slice:
			addMethods(u.Elem())
		case *types.Array:
			addMethods(u.Elem())
		case *types.Map:
			addMethods(u.Key())
			addMethods(u.Elem())
		case *types.Chan:
			addMethods(u.Elem())
		case *types.Struct:
			for i := 0; i < u.NumFields(); i++ {
				addMethods(u.Field(i).Type())
			}
		}
	}

	for f := range all {
		if root(f) {
			add(f)
		}
	}
	for len(queue) > 0 {
		fn := queue[0]
		queue = queue[1:]
		for _, anon := range fn.AnonFuncs {
			add(anon)
		}
		for _, b := range fn.Blocks {
			for _, instr := range b.Instrs {
				if call, ok := instr.(ssa.CallInstruction); ok {
					common := call.Common()
					if common.IsInvoke() {
						iface := common.Value.Type().Underlying().(*types.Interface)
						for _, m := range methodsByID[common.Method.Id()] {
							if types.Implements(m.Signature.Recv().Type(), iface) {
								add(m)
							}
						}
					} else if callee := common.StaticCallee(); callee != nil {
						add(callee)
						if o := callee.Origin(); o != nil && o.Name() == "TypeFor" && pkgPathOf(o) == "reflect" {
							for _, ta := range callee.TypeArgs() {
								addMethods(ta)
							}
						}
					}
				}
				if mi, ok := instr.(*ssa.MakeInterface); ok && !typeOnly(mi, fn) {
					addMethods(mi.X.Type())
				}
				for _, op := range instr.Operands(nil) {
					if op == nil {
						continue
					}
					if f, ok := (*op).(*ssa.Function); ok {
						add(f)
					}
				}
			}
		}
	}
	return seen
}
