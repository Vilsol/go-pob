package calculator

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

var modParserExtMarkerRe = regexp.MustCompile(`^\s*//gopob:ext (\S+)(?: (.*))?$`)

func modParserExtMarkers(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open("mod_parser_ext.go")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	markers := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := modParserExtMarkerRe.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		if _, dup := markers[m[1]]; dup {
			t.Errorf("//gopob:ext %s appears twice", m[1])
		}
		markers[m[1]] = strings.TrimSpace(m[2])
	}
	return markers
}

func requireModParserExtOff(t *testing.T) {
	t.Helper()
	if modParserExtEnabled.Load() {
		t.Skip("NOT-RUN: the extension layer is on (GOPOB_MOD_EXT=1); this test reads the PoB-grammar parse")
	}
}

// TestModParserExtRegistry: Every extension row has a `//gopob:ext <id> <reason>` marker and an example; every marker names a row.
func TestModParserExtRegistry(t *testing.T) {
	markers := modParserExtMarkers(t)
	ids := map[string]bool{}
	for _, ext := range modParserExtList {
		if ids[ext.id] {
			t.Errorf("extension id %q is used twice", ext.id)
		}
		ids[ext.id] = true
		if reason, ok := markers[ext.id]; !ok || reason == "" {
			t.Errorf("extension %q has no `//gopob:ext %s <reason>` marker in mod_parser_ext.go", ext.id, ext.id)
		}
		if len(ext.examples) == 0 {
			t.Errorf("extension %q has no example line, so the pin-bump tripwire cannot watch it", ext.id)
		}
	}
	for id := range markers {
		if !ids[id] {
			t.Errorf("//gopob:ext %s names no row in modParserExtList", id)
		}
	}
}

// TestModParserExtTripwire: Pin-bump tripwire, port side: an extension example the PoB-grammar port parses means the row is dead.
func TestModParserExtTripwire(t *testing.T) {
	requireModParserExtOff(t)
	for _, ext := range modParserExtList {
		for _, ex := range ext.examples {
			if id, _ := parseModExt(ex); id != ext.id {
				t.Errorf("extension %q: its example %q lands on %q", ext.id, ex, id)
			}
			if e := ParseMod(ex, false); !modParserExtUnsupported(e) {
				t.Errorf("extension %q: the PoB-grammar port now parses %q whole — retire the extension", ext.id, ex)
			}
		}
	}
}

// TestModParserExtTripwirePin: Pin-bump tripwire, pin side: an example the pin's ModCache parses whole means PoB has caught up.
func TestModParserExtTripwirePin(t *testing.T) {
	whole := map[string]bool{}
	mcdEach(t, func(key string, _ []mcdMod, pobWhole bool) { whole[key] = pobWhole })
	if len(whole) < 10000 {
		t.Fatalf("REFUSING: only %d ModCache keys read", len(whole))
	}
	if !whole["+10 to maximum Life"] {
		t.Fatal("REFUSING: the control line is not whole in the cache — the reader is broken")
	}
	for _, ext := range modParserExtList {
		for _, ex := range ext.examples {
			if whole[ex] {
				t.Errorf("extension %q: the pin's ModCache parses %q whole — PoB caught up, retire the extension", ext.id, ex)
			}
		}
	}
}

// TestModParserExtCoversGoOnly: The census's go-only class: a ModCache line the pin rejects and we accept must come from a registered extension.
func TestModParserExtCoversGoOnly(t *testing.T) {
	requireModParserExtOff(t)
	markers := modParserExtMarkers(t)
	rejected, viaExt := 0, 0
	mcdEach(t, func(key string, _ []mcdMod, pobWhole bool) {
		if pobWhole {
			return
		}
		rejected++
		if !modParserExtUnsupported(ParseMod(key, false)) {
			t.Errorf("go-only %q: the PoB-grammar port accepts a line the pin rejects — move the row to mod_parser_ext.go behind a //gopob:ext marker, or remove it", key)
			return
		}
		if id, _ := parseModExt(key); id != "" {
			viaExt++
			if markers[id] == "" {
				t.Errorf("go-only %q comes from extension %q, which has no //gopob:ext marker", key, id)
			}
		}
	})
	if rejected < 1000 {
		t.Fatalf("REFUSING: only %d lines the pin rejects — the reader is broken", rejected)
	}
	t.Logf("lines the pin rejects: %d; accepted by a registered extension: %d", rejected, viaExt)
}
