package calculator2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/data/poe2"
)

type tradeHashOracle struct {
	Value   *float64 `json:"value"`
	Negate  any      `json:"negate"`
	Line    string   `json:"line"`
	Options []struct {
		Value *float64 `json:"value"`
		Type  string   `json:"type"`
		ID    string   `json:"id"`
	} `json:"options"`
	Hashes []uint32 `json:"hashes"`
}

// TradeHelpers' option and stat description matches agree with PoB-PoE2's on every TRADE_STRIDE-th (default 5) line
// of the item mod tables (testdata/trade_hash.lua).
func TestTradeMatchesPoB(t *testing.T) {
	if os.Getenv("OPERATOR") != "true" {
		t.Skip("SKIP: runs the PoB-PoE2 pin under luajit; set OPERATOR=true")
	}
	if _, err := poe2.TradeStats(); err != nil {
		t.Skipf("SKIP: %v", err)
	}
	if _, err := poe2.StatDescriptions(); err != nil {
		t.Skipf("SKIP: %v", err)
	}
	pob, luajit := pobPoE2Probe(t)
	probe, err := filepath.Abs(filepath.Join("testdata", "trade_hash.lua"))
	if err != nil {
		t.Fatal(err)
	}
	stride := os.Getenv("TRADE_STRIDE")
	if stride == "" {
		stride = "5"
	}
	out := filepath.Join(t.TempDir(), "trade.jsonl")
	cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe, out, stride)
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the probe failed: %v\n%s", err, stderr.String())
	}
	b, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	lines, bad, hashed, options := 0, 0, 0, 0
	for sc.Scan() {
		var want tradeHashOracle
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		lines++
		for _, modType := range []string{"explicit", "implicit", "enchant"} {
			var wantID string
			var wantValue *float64
			for _, o := range want.Options {
				if o.Type == modType {
					wantID, wantValue = o.ID, o.Value
				}
			}
			id, value := findTradeIDOption(want.Line, modType)
			if wantID != "" {
				options++
			}
			if id != wantID || !reflect.DeepEqual(value, wantValue) {
				if bad++; bad <= 20 {
					t.Errorf("%q as %s: option %q %v, PoB-PoE2 %q %v", want.Line, modType, id, fmtp(value), wantID, fmtp(wantValue))
				}
			}
		}
		hashes, value, negate := findTradeHash(want.Line)
		if len(want.Hashes) > 0 {
			hashed++
		}
		gotNegate := any("nil")
		if negate != nil {
			gotNegate = *negate
		}
		if !reflect.DeepEqual(append([]uint32{}, hashes...), append([]uint32{}, want.Hashes...)) || !reflect.DeepEqual(value, want.Value) || gotNegate != want.Negate {
			if bad++; bad <= 20 {
				t.Errorf("%q: hashes %v value %v negate %v, PoB-PoE2 %v %v %v", want.Line, hashes, fmtp(value), gotNegate, want.Hashes, fmtp(want.Value), want.Negate)
			}
		}
	}
	t.Logf("%d lines, %d with trade hashes, %d option matches, %d disagree", lines, hashed, options, bad)
	if hashed == 0 || bad > 0 {
		t.Fail()
	}
}

func fmtp(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}
