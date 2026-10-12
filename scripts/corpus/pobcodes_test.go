package main

import "testing"

// TestPobCodesIDShape pins the length test against ids VERIFIED with the live API rather than eyeballed.
func TestPobCodesIDShape(t *testing.T) {
	spec := hostSpecFor("pob.codes")
	if spec == nil {
		t.Fatal("no hostSpec for pob.codes")
	}
	for id, want := range map[string]bool{
		"1MQDtBmrceQ": true,
		"A27ugRO_BwG": true,
		"WegUAPuN_oW": true,
		"iUJWTlN-p-N": true,
		"A27ugRO":     false,
		"WegUAPuN":    false,
		"":            false,
	} {
		if got := spec.validID(id); got != want {
			t.Errorf("validID(%q) = %v, want %v", id, got, want)
		}
	}
}

// TestPobCodesExtraction proves the pattern reaches the extractor, and that a truncated capture is REJECTED rather than silently indexed.
func TestPobCodesExtraction(t *testing.T) {
	found, rejected := extractLinks("check https://pob.codes/b/1MQDtBmrceQ and https://pob.codes/b/A27ugRO")
	var ids []string
	for _, f := range found {
		if f.Host == "pob.codes" {
			ids = append(ids, f.ID)
		}
	}
	if len(ids) != 1 || ids[0] != "1MQDtBmrceQ" {
		t.Errorf("extracted %v, want exactly [1MQDtBmrceQ]", ids)
	}
	if rejected["pob.codes"] != 1 {
		t.Errorf("the truncated id was not COUNTED as an id-shape rejection: %v", rejected)
	}
}

// TestPobCodesUsesItsPermittedRawEndpoint.
func TestPobCodesUsesItsPermittedRawEndpoint(t *testing.T) {
	pc := hostSpecFor("pob.codes")
	if !pc.rawAllowed {
		t.Error("pob.codes should use its raw endpoint by default — robots.txt permits it")
	}
	if got := pc.raw("1MQDtBmrceQ"); got != "https://api.pob.codes/1MQDtBmrceQ/raw" {
		t.Errorf("raw() = %s", got)
	}

	for _, h := range []string{"pastebin.com", "pobb.in"} {
		if hostSpecFor(h).rawAllowed {
			t.Errorf("%s must not default to its raw endpoint — robots.txt disallows it", h)
		}
	}
}
