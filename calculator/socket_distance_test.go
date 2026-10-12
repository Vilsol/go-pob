package calculator

import (
	"testing"
)

// TestSocketDistanceToClassStart: PassiveTreeView.lua:1494-1497.
func TestSocketDistanceToClassStart(t *testing.T) {
	build := loadCorpusBuild(t, "0052-juggernaut-cleave.xml")
	m := GetMiscCalculator(build)
	alloc := allocSet(build, specTree())
	near, far := 0, 0
	for _, s := range treeSockets(specTree()) {
		d := m.SocketDistanceToClassStart(int(s))
		switch {
		case !alloc[s] && d != 0:
			t.Errorf("unallocated socket %d: distance %d", s, d)
		case alloc[s] && d > 0:
			near++
			far = max(far, d)
		}
	}
	if near == 0 || far < 3 {
		t.Fatalf("%d allocated sockets with a distance, the farthest %d", near, far)
	}
}
