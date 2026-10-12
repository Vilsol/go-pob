package calculator

import (
	"encoding/json"
	"os"
	"testing"
)

// TestDumpItemModLineOrder is the OUR-side half of the mod-line ordering differential whose pin-side half is oracle/item_modline_order.lua.
func TestDumpItemModLineOrder(t *testing.T) {
	in, out := os.Getenv("MODLINE_ORDER_IN"), os.Getenv("MODLINE_ORDER_OUT")
	if in == "" || out == "" {
		t.Skip("NOT-RUN: set MODLINE_ORDER_IN and MODLINE_ORDER_OUT. This has NOT told you the " +
			"two sides agree -- it has told you it did not look.")
	}

	raw, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name string `json:"name"`
		Raw  string `json:"raw"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("REFUSING: cases file is empty -- an empty run must not look like agreement")
	}

	type line struct {
		Index int      `json:"index"`
		Line  string   `json:"line"`
		Range *float64 `json:"range"`
		Extra bool     `json:"extra"`
	}
	results := make([]map[string]any, 0, len(cases))
	total := 0
	for _, c := range cases {
		it := ParseItem(c.Raw)
		lists := map[string][]line{}
		add := func(name string, mls []ModLine) {
			ls := make([]line, 0, len(mls))
			for k, ml := range mls {
				ls = append(ls, line{Index: k + 1, Line: ml.Line, Range: ml.Range, Extra: ml.Extra != ""})
				total++
			}
			lists[name] = ls
		}
		add("buffModLines", nil)
		add("enchantModLines", it.EnchantModLines)
		add("scourgeModLines", it.ScourgeModLines)
		add("implicitModLines", it.ImplicitModLines)
		add("explicitModLines", it.ExplicitModLines)
		add("crucibleModLines", nil)
		results = append(results, map[string]any{"name": c.Name, "lists": lists})
	}
	if total == 0 {
		t.Fatalf("REFUSING: parsed %d cases and produced ZERO mod lines", len(results))
	}

	blob, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, blob, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote mod-line order for %d items (%d lines) to %s", len(results), total, out)
}
