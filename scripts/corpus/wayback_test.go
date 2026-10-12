package main

import (
	"strings"
	"testing"
)

const cdxGood = `https://pobarchives.com/build/bLtXe6t6 20260804141513
https://pobarchives.com/build/cXNKad8n 20260804141513
https://pobarchives.com/build/yQhnNLAp 20260731090102
`

// TestParseCDXRefusesAnErrorPage is the assertion this arm exists behind.
func TestParseCDXRefusesAnErrorPage(t *testing.T) {
	for _, body := range []string{
		"<html><body><h1>503 Service Unavailable</h1>\nNo server is available to handle this request.\n</body></html>",
		"<!DOCTYPE html>\n<html><head><title>Wayback Machine</title></head><body>Blocked</body></html>",
	} {
		if _, err := parseCDXRows([]byte(body)); err == nil {
			t.Errorf("parsed an error page as an index instead of refusing: %.60s", body)
		}
	}

	rows, err := parseCDXRows([]byte(cdxGood))
	if err != nil {
		t.Fatalf("refused a valid CDX body: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("parsed %d rows from a 3-row body", len(rows))
	}
	if rows[0].original != "https://pobarchives.com/build/bLtXe6t6" || rows[0].timestamp != "20260804141513" {
		t.Errorf("row 0 = %+v", rows[0])
	}
}

// TestParseCDXRejectsAShiftedColumn guards the `fl=` field order.
func TestParseCDXRejectsAShiftedColumn(t *testing.T) {
	shifted := "20260804141513 https://pobarchives.com/build/bLtXe6t6\n"
	rows, err := parseCDXRows([]byte(shifted))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("accepted a row whose second field is not a 14-digit timestamp: %+v", rows)
	}

	extra := "https://pobarchives.com/build/bLtXe6t6 20260804141513 text/html\n"
	if rows, _ := parseCDXRows([]byte(extra)); len(rows) != 0 {
		t.Errorf("accepted a 3-column row: %+v", rows)
	}
}

// TestWaybackReplayIsTheRewrittenForm pins the URL shape.
func TestWaybackReplayIsTheRewrittenForm(t *testing.T) {
	got := waybackReplayURL(waybackSnapshot{original: "https://pobarchives.com/build/339ywXFi", timestamp: "20260221172715"})
	want := "https://web.archive.org/web/20260221172715/https://pobarchives.com/build/339ywXFi"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if strings.Contains(got, "id_") {
		t.Error("the replay URL uses the id_ form, whose Content-Encoding Go will not decode")
	}
}

func TestSnapshotTimeIsUTC(t *testing.T) {
	at, err := snapshotTime("20260804141513")
	if err != nil {
		t.Fatalf("snapshotTime: %v", err)
	}
	if got := at.UTC().Format("2006-01-02T15:04:05Z"); got != "2026-08-04T14:15:13Z" {
		t.Errorf("got %s", got)
	}
	if _, err := snapshotTime("not-a-timestamp"); err == nil {
		t.Error("accepted an unparseable timestamp")
	}
}

// TestHostOfNamesTheArchivedHost keeps provenance pointing at where a build actually came from.
func TestHostOfNamesTheArchivedHost(t *testing.T) {
	for in, want := range map[string]string{
		"https://pobarchives.com/build/x": "pobarchives.com",
		"https://www.pobarchives.com/b":   "pobarchives.com",
		"not a url":                       "unknown",
	} {
		if got := hostOf(in); got != want {
			t.Errorf("hostOf(%q) = %q, want %q", in, got, want)
		}
	}
}
