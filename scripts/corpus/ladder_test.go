package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestIntervalFromPolicy pins the reading of GGG's published limits.
func TestIntervalFromPolicy(t *testing.T) {
	for header, want := range map[string]time.Duration{
		"30:60:120,90:1800:600,180:7200:3600": 40 * time.Second,
		"5:5:10,10:10:30,15:10:300":           time.Second,
		"10:60:60":                            6 * time.Second,
		"":                                    0,
		"not:a:policy":                        0,
		"0:60:0":                              0,
		"10:0:0":                              0,
	} {
		if got := intervalFromPolicy(header); got != want {
			t.Errorf("intervalFromPolicy(%q) = %v, want %v", header, got, want)
		}
	}
}

// TestEnsureAtLeastNeverSpeedsUp.
func TestEnsureAtLeastNeverSpeedsUp(t *testing.T) {
	hl := newHostLimiter(time.Second, map[string]time.Duration{"slow.example": 44 * time.Second})

	hl.ensureAtLeast("slow.example", time.Second)
	if got := hl.interval["slow.example"]; got != 44*time.Second {
		t.Errorf("a looser policy lowered the interval to %v", got)
	}
	hl.ensureAtLeast("slow.example", 90*time.Second)
	if got := hl.interval["slow.example"]; got != 90*time.Second {
		t.Errorf("a tighter policy was ignored: %v", got)
	}
	hl.ensureAtLeast("slow.example", 0)
	if got := hl.interval["slow.example"]; got != 90*time.Second {
		t.Errorf("a zero policy changed the interval to %v", got)
	}
}

func TestCharacterIDIsFilesystemSafe(t *testing.T) {
	for in, want := range map[[2]string]string{
		{"Poteitik#3151", "ПОТЕЙТИК"}: "Poteitik_3151.ПОТЕЙТИК",
		{"a b#1", "c/d"}:              "a_b_1.c_d",
	} {
		if got := characterID(in[0], in[1]); got != want {
			t.Errorf("characterID(%q,%q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

// TestGetWithRetryNamesAGoneCharacter: the ladder checkpoints a 404 instead of retrying it on every restart.
func TestGetWithRetryNamesAGoneCharacter(t *testing.T) {
	code := http.StatusNotFound
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}))
	defer srv.Close()
	c := newClient(newHostLimiter(0, nil), "test", false)

	_, err := getWithRetry(t.Context(), c, "example.test", srv.URL, 1, false, false)
	if !errors.Is(err, errFetchNotFound) {
		t.Errorf("a 404 returned %v, want errFetchNotFound", err)
	}
	code = http.StatusForbidden
	_, err = getWithRetry(t.Context(), c, "example.test", srv.URL, 1, false, false)
	if err == nil || errors.Is(err, errFetchNotFound) {
		t.Errorf("a 403 returned %v, want a retryable failure that is not errFetchNotFound", err)
	}
}
