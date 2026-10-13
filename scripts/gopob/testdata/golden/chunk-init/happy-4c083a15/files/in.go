package raw

type V struct {
	A string
	B int
}

var Table = buildTable()

func buildTable() map[string]V {
	m := make(map[string]V, 10)
	tablePart0(m)
	tablePart1(m)
	tablePart2(m)
	tablePart3(m)
	return m
}

func tablePart0(m map[string]V) {
	// src/Data/Skills/act_str.lua:1201
	m["plain"] = V{A: "x", B: 1}
	m["brace in string"] = V{A: "}", B: 2}
	m["open in string"] = V{A: "{", B: 3}
}

func tablePart1(m map[string]V) {
	// a comment with } } in it
	m["after comment"] = V{A: "y", B: 4}
	m["block comment"] = /* } } */ {A: "z", B: 5}
	m["raw string"] = V{A: `raw } } raw`, B: 6}
}

func tablePart2(m map[string]V) {
	m["escaped quote"] = V{A: "he said \"}\" loudly", B: 7}
	m["colon: in key"] = V{A: "v:w", B: 8}
	m["rune"] = V{A: string('}'), B: 9}
}

func tablePart3(m map[string]V) {
	m["nested"] = V{A: "n", B: len([]int{1, 2, 3})}
}


func Unrelated() int { return 1 }
