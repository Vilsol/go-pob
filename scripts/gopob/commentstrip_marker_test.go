package main

import "testing"

func TestCommentStripKeepsRetiredVersionMarker(t *testing.T) {
	for _, text := range []string{"//go-only: retired-version-ok: a DIFFERENT version is the control", "//retired-version-ok: x"} {
		if got := csClassify(text, nil); got != csMarker {
			t.Errorf("%q classified %s, want marker", text, csClassNames[got])
		}
	}
	if got := csClassify("// a retired version is fine here", nil); got == csMarker {
		t.Errorf("plain prose classified as a marker")
	}
}

func TestCommentStripKeepsCrystallineDirective(t *testing.T) {
	for _, text := range []string{"//crystalline:exports", "//crystalline:export"} {
		if got := csClassify(text, nil); got != csMarker {
			t.Errorf("%q classified %s, want marker", text, csClassNames[got])
		}
	}
}

func TestCommentStripKeepsFoldDeadMarker(t *testing.T) {
	if got := csClassify("//fold-dead-ok: same as the extractor above", nil); got != csMarker {
		t.Errorf("fold-dead-ok classified %s, want marker", csClassNames[got])
	}
}

func TestCommentStripKeepsModParserExtMarker(t *testing.T) {
	if got := csClassify("//gopob:ext adrenaline-on-kill-singular-second PoB's row needs the plural", nil); got != csMarker {
		t.Errorf("gopob:ext classified %s, want marker", csClassNames[got])
	}
}
