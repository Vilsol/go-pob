package main

import (
	"fmt"
	"strings"
)

type dmCheck struct {
	name string
	ok   bool
}

type dmChecker struct{ checks []dmCheck }

func (c *dmChecker) add(name string, fn func() bool) {
	ok := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				ok = false
				name = fmt.Sprintf("%s (raised %v)", name, r)
			}
		}()
		ok = fn()
	}()
	c.checks = append(c.checks, dmCheck{name, ok})
}

func dmFakeSpec(goName, mode string, subseq ...string) dmSpec {
	if subseq == nil {
		subseq = []string{}
	}
	return dmSpec{goName: goName, mode: mode, subseq: subseq}
}

func dmRefuses(fn func() error, wantSourceMissing bool, needle string) bool {
	err := fn()
	if err == nil {
		return false
	}
	isSM := dmExcName(err) == "SourceMissing"
	if isSM != wantSourceMissing {
		return false
	}
	return strings.Contains(err.Error(), needle)
}

func dmParseOrNil(src string) any {
	v, _, err := ldParseValue(src, 0)
	if err != nil {
		return nil
	}
	return v
}

func dmSelftest() int {
	c := &dmChecker{}

	c.add("DIFFERS on a changed value", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact"),
			map[string]any{"a": map[string]any{"x": 1}},
			map[string]any{"a": map[string]any{"x": 2}})
		return v == "DIFFERS"
	})
	c.add("DIFFERS on a row only we have", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact"),
			map[string]any{"a": 1, "b": 1}, map[string]any{"a": 1})
		return v == "DIFFERS"
	})
	c.add("DIFFERS on a row only PoB has (mode=exact)", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact"),
			map[string]any{"a": 1}, map[string]any{"a": 1, "b": 1})
		return v == "DIFFERS"
	})
	c.add("IDENTICAL ignores PoB-only rows under mode=subset", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "subset"),
			map[string]any{"a": 1}, map[string]any{"a": 1, "b": 1})
		return v == "IDENTICAL"
	})
	c.add("subseq accepts a dropped mod", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact", "mods"),
			map[string]any{"a": map[string]any{"mods": []any{"X"}}},
			map[string]any{"a": map[string]any{"mods": []any{"X", "Y"}}})
		return v == "IDENTICAL"
	})
	c.add("subseq rejects a REORDERED mod list", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact", "mods"),
			map[string]any{"a": map[string]any{"mods": []any{"Y", "X"}}},
			map[string]any{"a": map[string]any{"mods": []any{"X", "Y"}}})
		return v == "DIFFERS"
	})
	c.add("subseq rejects a mod PoB does not have", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact", "mods"),
			map[string]any{"a": map[string]any{"mods": []any{"Z"}}},
			map[string]any{"a": map[string]any{"mods": []any{"X", "Y"}}})
		return v == "DIFFERS"
	})

	dmDeviations[dmDevKey{"Fake", "a"}] = "selftest"
	c.add("a DECLARED deviation is not counted as DIFFERS", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact"),
			map[string]any{"a": 1}, map[string]any{"a": 2})
		return v == "IDENTICAL"
	})
	c.add("a STALE deviation (declared, no longer differing) FAILS", func() bool {
		v, _, _, _ := dmCompare(dmFakeSpec("Fake", "exact"),
			map[string]any{"a": 1}, map[string]any{"a": 1})
		return v == "DIFFERS"
	})
	delete(dmDeviations, dmDevKey{"Fake", "a"})

	c.add("a DECLARED unmapped table carries a reason", func() bool {
		for _, t := range dmTables() {
			if t.unmapped != "" && strings.TrimSpace(t.unmapped) == "" {
				return false
			}
		}
		return true
	})
	c.add("every table is either extractable or DECLARED unmapped", func() bool {
		for _, t := range dmTables() {
			if t.extract == nil && t.unmapped == "" {
				return false
			}
		}
		return true
	})
	c.add("a missing source file raises SourceMissing", func() bool {
		return dmRefuses(func() error {
			_, err := dmRead("/nonexistent-pob", "src/Data/Gems.lua")
			return err
		}, true, "")
	})
	c.add("a missing skills glob raises SourceMissing", func() bool {
		return dmRefuses(func() error {
			_, err := dmSkillFiles("/nonexistent-pob")
			return err
		}, true, "matched nothing")
	})
	c.add("the parser REFUSES a value it cannot read", func() bool {
		return dmRefuses(func() error {
			_, err := dmFlagSet("not a table")
			return err
		}, false, "expected a table")
	})
	c.add("the parser REFUSES a non-mod entry in a mod list", func() bool {
		return dmRefuses(func() error {
			_, err := dmModCallNames(dmTableOf([]any{1, 2}))
			return err
		}, false, "not a mod call")
	})
	for _, tc := range []struct{ what, text string }{
		{"a long string", "\nskills[\"A\"] = { x = [[\nskills[\"B\"] = { y = 1 }\n]] }\n" +
			"skills[\"C\"] = { z = 3 }\n"},
		{"a block comment", "\nskills[\"A\"] = { x = 1, --[[\nskills[\"B\"] = { y = 1 }\n]] z = 2 }\n" +
			"skills[\"C\"] = { z = 3 }\n"},
	} {
		text := tc.text
		c.add(fmt.Sprintf("scan_assignments REFUSES an entry that %s runs past the next one", tc.what),
			func() bool {
				return dmRefuses(func() error {
					_, err := ldScanAssignments(text, "skills")
					return err
				}, false, "parsed past the next entry")
			})
	}
	c.add("scan_assignments REFUSES an unterminated table, and says so", func() bool {
		return dmRefuses(func() error {
			_, err := ldScanAssignments("\nskills[\"A\"] = { x = { \n\nskills[\"B\"] = { y = 1 }\n", "skills")
			return err
		}, false, "unterminated table")
	})
	c.add("a function body does not truncate its table", func() bool {
		rows, err := ldScanAssignments("\nskills[\"A\"] = {\n f = function(a) if a then return {1} end end,\n"+
			" parts = { { name = \"P\" } },\n}\n", "skills")
		if err != nil || len(rows) == 0 {
			return false
		}
		parts, ok := rows[0].table.hash["parts"].(*ldTable)
		if !ok || len(parts.array) == 0 {
			return false
		}
		first, ok := parts.array[0].(*ldTable)
		return ok && dmHashStr(first, "name") == "P"
	})
	c.add("Lua duplicate keys resolve LAST-WINS", func() bool {
		t, ok := dmParseOrNil("{ a = 1, a = 2 }").(*ldTable)
		if !ok {
			return false
		}
		return t.hash["a"] == 2
	})
	for _, tc := range []struct {
		nm    string
		fn    func(any) (any, error)
		okIn  string
		okOut []any
	}{
		{"skillTypes", dmSkillTypeList, "{ SkillType.Spell }", []any{"Spell"}},
		{"baseMods", dmModCallNames, `{ mod("Damage", "BASE", 1) }`, []any{"Damage"}},
	} {
		fn, okIn, okOut := tc.fn, tc.okIn, tc.okOut
		c.add(fmt.Sprintf("%s REFUSES an alias to another skill's table", tc.nm), func() bool {
			return dmRefuses(func() error {
				_, err := fn(&ldExpr{text: "skills.Other.x"})
				return err
			}, false, "expected a table")
		})
		c.add(fmt.Sprintf("%s still reads an ordinary table", tc.nm), func() bool {
			got, err := fn(dmParseOrNil(okIn))
			return err == nil && dmEqual(got, dmAnyList(okOut))
		})
	}
	c.add("minion rows REFUSE an alias to another minion's table", func() bool {
		return dmRefuses(func() error {
			_, err := dmMinionRow(&ldExpr{text: "minions.Other"}, "")
			return err
		}, false, "expected a table")
	})
	c.add("minion rows still read an ordinary table", func() bool {
		row, err := dmMinionRow(dmParseOrNil(`{ name = "M", life = 2 }`), "")
		if err != nil {
			return false
		}
		m, ok := row.(map[string]any)
		return ok && m["name"] == "M"
	})
	c.add("parts REFUSES an alias to another skill's table", func() bool {
		return dmRefuses(func() error {
			_, err := dmPartsRows(&ldExpr{text: "skills.ExplosiveTrap.parts"})
			return err
		}, false, "expected a table")
	})
	c.add("parts still reads an ordinary parts table", func() bool {
		got, err := dmPartsRows(dmParseOrNil(`{ { name = "P", spell = true } }`))
		if err != nil {
			return false
		}
		want := []any{map[string]any{"name": "P", "flags": map[string]any{"spell": true}}}
		return dmEqual(got, want)
	})

	width := 0
	for _, ch := range c.checks {
		if n := len([]rune(ch.name)); n > width {
			width = n
		}
	}
	failed := 0
	for _, ch := range c.checks {
		mark := "FAIL"
		if ch.ok {
			mark = "OK  "
		} else {
			failed++
		}
		fmt.Printf("  %s  %s\n", mark, dmPad(ch.name, width))
	}
	fmt.Printf("\nselftest: %d/%d checks reachable\n", len(c.checks)-failed, len(c.checks))
	if failed > 0 {
		return 1
	}
	return 0
}

func dmPad(s string, width int) string {
	n := len([]rune(s))
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func dmTableOf(items []any) *ldTable {
	t := newLDTable()
	t.array = items
	return t
}

func dmAnyList(xs []any) []any { return xs }
