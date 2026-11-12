package main

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func committed(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", path)) //nolint:gosec // the generator's own outputs
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func pngSide(t *testing.T, what string, b []byte) int {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != h {
		t.Fatalf("%s: %dx%d is not square", what, w, h)
	}
	return img.Bounds().Dx()
}

func TestDesktopIconPNG(t *testing.T) {
	t.Parallel()
	if n := pngSide(t, "desktop/icon.png", committed(t, "desktop/icon.png")); n != 512 {
		t.Errorf("desktop/icon.png is %d px, want 512", n)
	}
}

func TestICOEntriesDecode(t *testing.T) {
	t.Parallel()
	for path, want := range icos {
		b := committed(t, path)
		if r, typ := binary.LittleEndian.Uint16(b), binary.LittleEndian.Uint16(b[2:]); r != 0 || typ != 1 {
			t.Fatalf("%s: header %d %d, want 0 1", path, r, typ)
		}
		var got []int
		for i := range int(binary.LittleEndian.Uint16(b[4:])) {
			e := b[6+16*i:]
			size, off := binary.LittleEndian.Uint32(e[8:]), binary.LittleEndian.Uint32(e[12:])
			n := pngSide(t, path, b[off:off+size])
			if dim := int(e[0]); dim != n%256 || e[1] != e[0] {
				t.Errorf("%s: directory says %dx%d for a %d px PNG", path, e[0], e[1], n)
			}
			got = append(got, n)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: sizes %v, want %v", path, got, want)
		}
	}
}

func TestICNSEntriesDecode(t *testing.T) {
	t.Parallel()
	b := committed(t, icnsPath)
	if string(b[:4]) != "icns" || int(binary.BigEndian.Uint32(b[4:])) != len(b) {
		t.Fatalf("icns header %q %d for %d bytes", b[:4], binary.BigEndian.Uint32(b[4:]), len(b))
	}
	got := map[string]int{}
	for at := 8; at < len(b); {
		typ, n := string(b[at:at+4]), int(binary.BigEndian.Uint32(b[at+4:]))
		got[typ] = pngSide(t, typ, b[at+8:at+n])
		at += n
	}
	if len(got) != len(icnsTypes) {
		t.Errorf("icns has %d entries, want %d", len(got), len(icnsTypes))
	}
	for _, e := range icnsTypes {
		if got[e.typ] != e.size {
			t.Errorf("icns %s is %d px, want %d", e.typ, got[e.typ], e.size)
		}
	}
}
