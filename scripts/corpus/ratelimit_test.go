package main

import (
	"testing"
	"time"
)

// TestSlowDownNeverSpeedsAHostUp is the only-ever-increases promise, tested at the boundary where it used to break.
func TestSlowDownNeverSpeedsAHostUp(t *testing.T) {
	hl := newHostLimiter(350*time.Millisecond, map[string]time.Duration{
		"archive.example": 5 * time.Second,
		"fast.example":    350 * time.Millisecond,
	})

	if got := hl.slowDown("archive.example", 4*time.Second); got < 5*time.Second {
		t.Errorf("slowDown returned %v for a host already at 5s — the ceiling clamped it DOWN", got)
	}

	first := hl.slowDown("fast.example", 4*time.Second)
	if first != 700*time.Millisecond {
		t.Errorf("first shed gave %v, want 700ms (doubling from 350ms)", first)
	}
	if second := hl.slowDown("fast.example", 4*time.Second); second != 1400*time.Millisecond {
		t.Errorf("second shed gave %v, want 1.4s", second)
	}
}

// TestRateShedCodes pins which statuses mean "too fast".
func TestRateShedCodes(t *testing.T) {
	for code, want := range map[int]bool{
		429: true, 503: true,
		500: false, 502: false, 504: false, 403: false, 404: false, 200: false,
	} {
		if got := isRateShed(code); got != want {
			t.Errorf("isRateShed(%d) = %v, want %v", code, got, want)
		}
	}
}

// TestShedCounterIsPerHost pins the instrumentation that would have caught a crawl pretending to be healthy.
func TestShedCounterIsPerHost(t *testing.T) {
	c := newClient(newHostLimiter(time.Second, nil), "test", false)

	if got := c.shedCount("web.archive.org"); got != 0 {
		t.Fatalf("a fresh client reports %d sheds", got)
	}
	c.noteShed("web.archive.org")
	c.noteShed("web.archive.org")
	c.noteShed("pobb.in")

	if got := c.shedCount("web.archive.org"); got != 2 {
		t.Errorf("web.archive.org = %d, want 2", got)
	}
	if got := c.shedCount("pobb.in"); got != 1 {
		t.Errorf("pobb.in = %d, want 1", got)
	}
	if got := c.shedCount("pastebin.com"); got != 0 {
		t.Errorf("an untouched host reports %d sheds", got)
	}
}
