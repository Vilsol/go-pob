package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

//go:embed testdata/real-3_18-build.pobcode
var realBuildCode string

func decodeFixture(t *testing.T) []byte {
	t.Helper()
	x, err := decodeExport([]byte(realBuildCode))
	if err != nil {
		t.Fatalf("decoding the committed fixture: %v", err)
	}
	return x
}

func criteria318() criteria {
	return criteria{treeVersion: "3_18", minNodes: 80, minSlots: 8, slotMode: "occurrences"}
}

func TestAcceptsRealBuildOnItsOwnTree(t *testing.T) {
	v := judge(decodeFixture(t), criteria318())
	if !v.ok {
		t.Fatalf("a real 3.18 corpus build was rejected: %s %s", v.reason, v.detail)
	}
	if v.treeSeen != "3_18" || v.nodes < 80 || v.slots < 8 || v.stats == 0 {
		t.Fatalf("fixture read wrong: tree=%q nodes=%d slots=%d stats=%d",
			v.treeSeen, v.nodes, v.slots, v.stats)
	}
}

func TestRejectsRealBuildOnAnotherLeaguesTree(t *testing.T) {
	c := criteria318()
	c.treeVersion = "3_19"
	v := judge(decodeFixture(t), c)
	if v.ok || v.reason != rejWrongTree {
		t.Fatalf("want %s, got ok=%v reason=%s", rejWrongTree, v.ok, v.reason)
	}
	if v.detail != "3_18" {
		t.Fatalf("the rejection must NAME the version found, got %q", v.detail)
	}
}

// TestReadsTheActiveSpecNotTheFirst is the criterion that rejected 1066 of 3496 candidates in the 3.18 harvest.
func TestReadsTheActiveSpecNotTheFirst(t *testing.T) {
	x := string(decodeFixture(t))

	staleNodes := strings.Repeat("1,", 199) + "1"
	stale := `<Spec treeVersion="3_17" classId="0" ascendClassId="1" nodes="` + staleNodes + `"/>`

	twoSpecs := strings.Replace(x, `<Tree activeSpec="1">`, `<Tree activeSpec="2">`, 1)
	twoSpecs = strings.Replace(twoSpecs, "<Spec ", stale+"\n<Spec ", 1)

	if v := judge([]byte(twoSpecs), criteria318()); !v.ok {
		t.Fatalf("active spec 2 is 3_18 and must be accepted, got %s %s", v.reason, v.detail)
	}

	staleActive := strings.Replace(twoSpecs, `<Tree activeSpec="2">`, `<Tree activeSpec="1">`, 1)
	v := judge([]byte(staleActive), criteria318())
	if v.ok || v.reason != rejWrongTree || v.detail != "3_17" {
		t.Fatalf("active spec 1 is 3_17 and must be rejected as such, got ok=%v %s %s",
			v.ok, v.reason, v.detail)
	}
}

func TestRejectsBuildWithNoPlayerStat(t *testing.T) {
	x := string(decodeFixture(t))
	if !strings.Contains(x, "<PlayerStat ") {
		t.Fatal("fixture carries no PlayerStat, so this test cannot fail correctly")
	}
	stripped := removeAllElements(x, "<PlayerStat ")
	v := judge([]byte(stripped), criteria318())
	if v.ok || v.reason != rejNoPlayerStat {
		t.Fatalf("want %s, got ok=%v reason=%s", rejNoPlayerStat, v.ok, v.reason)
	}
}

func TestRejectsUnderEquippedCharacter(t *testing.T) {
	x := string(decodeFixture(t))
	stripped := removeAllElements(x, "<Slot ")
	v := judge([]byte(stripped), criteria318())
	if v.ok || v.reason != rejUnderEquipped {
		t.Fatalf("want %s, got ok=%v reason=%s", rejUnderEquipped, v.ok, v.reason)
	}
}

func TestRejectsStubTree(t *testing.T) {
	c := criteria318()
	c.minNodes = 100000
	v := judge(decodeFixture(t), c)
	if v.ok || v.reason != rejStubTree {
		t.Fatalf("want %s, got ok=%v reason=%s", rejStubTree, v.ok, v.reason)
	}
}

// TestThresholdsAreBinding guards against a criterion silently becoming a no-op.
func TestThresholdsAreBinding(t *testing.T) {
	x := decodeFixture(t)
	base := judge(x, criteria318())

	tight := criteria318()
	tight.minNodes = base.nodes + 1
	if judge(x, tight).ok {
		t.Error("minNodes one above the build's own count still accepted it")
	}

	tight = criteria318()
	tight.minSlots = base.slots + 1
	if judge(x, tight).ok {
		t.Error("minSlots one above the build's own count still accepted it")
	}

	exact := criteria318()
	exact.minNodes, exact.minSlots = base.nodes, base.slots
	if !judge(x, exact).ok {
		t.Error("thresholds set to exactly the build's own counts rejected it")
	}
}

// TestBlockedIsNeverAContentRejection is the single most important test here.
func TestBlockedIsNeverAContentRejection(t *testing.T) {
	interstitials := map[string][]byte{
		"cloudflare 200 challenge": []byte(`<!DOCTYPE html><html><head><title>Just a moment...</title></head>`),
		"attention required":       []byte(`<html><head><title>Attention Required! | Cloudflare</title>`),
		"js/cookie wall":           []byte(`<html><body>Enable JavaScript and cookies to continue</body></html>`),
		"unseen challenge page":    []byte(`<html><body>something we have never seen before</body></html>`),
	}
	for name, body := range interstitials {
		if got := classifyBody(body); got != fetchBlocked {
			t.Errorf("%s: classified %s, want %s — a block must never look like content", name, got, fetchBlocked)
		}
	}

	if got := classifyBody([]byte(realBuildCode)); got != fetchOK {
		t.Errorf("a real export code classified %s, want %s", got, fetchOK)
	}

	if got := classifyBody([]byte("Error, this paste has been removed.")); got != fetchNotCode {
		t.Errorf("a removed-paste body classified %s, want %s", got, fetchNotCode)
	}
}

func TestRetryabilityIsCorrectPerClass(t *testing.T) {
	terminal := map[fetchClass]bool{
		fetchOK: true, fetchNotFound: true, fetchBadRequest: true,
		fetchBlocked: false, fetchTransport: false, fetchNotCode: false,
	}
	for cls, want := range terminal {
		if got := cls.terminal(); got != want {
			t.Errorf("%s.terminal() = %v, want %v", cls, got, want)
		}
	}
}

// TestHTMLIsOnlyBlockedWhereItIsUnexpected pins both directions of the flag that decides whether an HTML body is an interception.
func TestHTMLIsOnlyBlockedWhereItIsUnexpected(t *testing.T) {
	page := []byte(`<!doctype html><html><body>an ordinary forum listing</body></html>`)
	if _, blocked := looksBlocked(page, false); !blocked {
		t.Error("HTML on a payload endpoint must be blocked")
	}
	if _, blocked := looksBlocked(page, true); blocked {
		t.Error("HTML where HTML is expected must NOT be blocked")
	}
	challenge := []byte(`<!doctype html><html><title>Just a moment...</title>`)
	if _, blocked := looksBlocked(challenge, true); !blocked {
		t.Error("a NAMED challenge page must be blocked even where HTML is expected")
	}
}

func TestExtractEmbeddedCodeFindsTheCodeInEachHostsPage(t *testing.T) {
	x := decodeFixture(t)
	pages := map[string]string{
		"pobb.in": `<html><body><textarea aria-label="Path of Building buildcode" readonly="">` +
			realBuildCode + `</textarea></body></html>`,
		"pastebin": `<html><body><ol class="text"><li class="li1"><div class="de1">` +
			realBuildCode + `</div></li></ol></body></html>`,
	}
	for host, page := range pages {
		got, ok := extractEmbeddedCode([]byte(page))
		if !ok {
			t.Fatalf("%s: no code extracted from its own page shape", host)
		}
		back, err := decodeExport(got)
		if err != nil || string(back) != string(x) {
			t.Fatalf("%s: extracted code did not round-trip to the fixture (err=%v)", host, err)
		}
	}
}

// TestExtractEmbeddedCodeRefusesNonPoBPayloads: The extractor must be unable to return the WRONG blob.
func TestExtractEmbeddedCodeRefusesNonPoBPayloads(t *testing.T) {
	decoy := `<html><script src="data:application/javascript;base64,` +
		strings.Repeat("QUJDRA", 400) + `"></script></html>`
	if _, ok := extractEmbeddedCode([]byte(decoy)); ok {
		t.Error("a long base64 decoy was accepted as a build")
	}
	if _, ok := extractEmbeddedCode([]byte(`<html><body>This paste has been removed.</body></html>`)); ok {
		t.Error("a page with no code at all returned a code")
	}
}

func TestDecodeExportToleratesPastebinLineWrapping(t *testing.T) {
	var wrapped strings.Builder
	for i := 0; i < len(realBuildCode); i += 76 {
		wrapped.WriteString(realBuildCode[i:min(i+76, len(realBuildCode))])
		wrapped.WriteString("\r\n")
	}
	got, err := decodeExport([]byte(wrapped.String()))
	if err != nil {
		t.Fatalf("wrapped code failed to decode: %v", err)
	}
	if string(got) != string(decodeFixture(t)) {
		t.Fatal("wrapped code decoded to different bytes")
	}
}

func TestLinkExtractionRejectsRatherThanEmittingDeadLinks(t *testing.T) {
	links, _ := extractLinks("build: https://pastebin.com/aBcD1234 mirror https://pobb.in/5v25ibooPzOx")
	if len(links) != 2 {
		t.Fatalf("want 2 links, got %d", len(links))
	}
	if _, rejected := extractLinks("https://pastebin.com/u/someuser"); rejected["pastebin.com"] == 0 {
		t.Error("a profile URL must be recorded as an id-shape rejection")
	}
	got, _ := extractLinks("see [pob](https://pastebin.com/aBcD1234), it works.")
	if len(got) != 1 || got[0].ID != "aBcD1234" {
		t.Errorf("markdown punctuation broke extraction: %+v", got)
	}
	got, _ = extractLinks("https://pastebin.com/raw/aBcD1234")
	if len(got) != 1 || got[0].ID != "aBcD1234" {
		t.Errorf("a /raw/ link must normalise to the same id: %+v", got)
	}
}

// TestUserScopedPobbinLinksParse pins the one pobb.in URL shape the extractor used to destroy.
func TestUserScopedPobbinLinksParse(t *testing.T) {
	const id = "aPkK1aLp6p53"
	for _, u := range []string{
		"https://pobb.in/" + id,
		"https://pobb.in/pob/" + id,
		"https://pobb.in/u/alice/" + id,
		"https://pobb.in/pob/u/alice/" + id,
		"https://pobb.in/u/some_user-1/" + id + "/raw",
	} {
		got, rejected := extractLinks("look: " + u)
		if len(got) != 1 || got[0].ID != id {
			t.Errorf("%s -> %+v (rejected %v), want the single id %q", u, got, rejected, id)
		}
	}
	if got, rejected := extractLinks("https://pobb.in/u/alice"); len(got) != 0 || rejected["pobb.in"] == 0 {
		t.Errorf("bare /u/<name> -> %+v (rejected %v), want 0 links and a counted rejection", got, rejected)
	}
	if a, b := hostSpecFor("pobb.in").canonical(id), "https://pobb.in/"+id; a != b {
		t.Errorf("canonical(%q) = %q, want %q", id, a, b)
	}
}

// TestMarkdownEscapedIDsSurvive pins the escape defect measured on 2026-08-11.
func TestMarkdownEscapedIDsSurvive(t *testing.T) {
	for _, tc := range []struct{ text, want string }{
		{`https://pobb.in/eVgBOQGYn\_JH`, "eVgBOQGYn_JH"},
		{`"description":"https://pobb.in/eVgBOQGYn\\_JH\n"`, "eVgBOQGYn_JH"},
		{`https://pobb.in/eyuzdX\_mh\_gg`, "eyuzdX_mh_gg"},
		{`https://pobb.in/u/some\_user/aPkK1aLp\_53`, "aPkK1aLp_53"},
	} {
		got, rejected := extractLinks(tc.text)
		if len(got) != 1 || got[0].ID != tc.want {
			t.Errorf("%s -> %+v (rejected %v), want the single id %q", tc.text, got, rejected, tc.want)
		}
	}
	if got, _ := extractLinks(`https://pobb.in/aPkK1aLp6p53\ and text`); len(got) != 1 || got[0].ID != "aPkK1aLp6p53" {
		t.Errorf("trailing backslash -> %+v, want aPkK1aLp6p53", got)
	}
	if got, _ := extractLinks("https://pobb.in/eVgBOQGYn_JH"); len(got) != 1 || got[0].ID != "eVgBOQGYn_JH" {
		t.Errorf("plain id -> %+v, want eVgBOQGYn_JH", got)
	}
}

func TestNamesBuildAfterItsMainSkill(t *testing.T) {
	doc, err := parsePoB(decodeFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.mainSkillLabel(); got != "absolution" {
		t.Errorf("mainSkillLabel() = %q, want %q (the fixture is 002-ascendant-absolution)", got, "absolution")
	}
	if got := doc.classLabel(); got != "ascendant" {
		t.Errorf("classLabel() = %q, want %q", got, "ascendant")
	}
}

func removeAllElements(x, prefix string) string {
	for {
		i := strings.Index(x, prefix)
		if i < 0 {
			return x
		}
		j := strings.Index(x[i:], ">")
		if j < 0 {
			return x
		}
		x = x[:i] + x[i+j+1:]
	}
}

func TestRateLimitIsWaitedOutRatherThanRecordedAsBlocked(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(realBuildCode))
	}))
	defer srv.Close()

	c := newClient(newHostLimiter(0, nil), "test", false)
	r := c.get(t.Context(), "example.test", srv.URL, false)

	if r.class != fetchOK {
		t.Fatalf("after two 429s the third attempt succeeds: got class %q (HTTP %d)", r.class, r.code)
	}
	if hits != 3 {
		t.Fatalf("want 3 attempts (two waited-out 429s then the win), got %d", hits)
	}
}

func TestRateLimitGivesUpAndStaysRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newClient(newHostLimiter(0, nil), "test", false)
	r := c.get(t.Context(), "example.test", srv.URL, false)

	if r.class != fetchBlocked {
		t.Fatalf("a host that never relents stays blocked: got %q", r.class)
	}
	if r.class.terminal() {
		t.Fatal("blocked must stay retryable so a resumed fetch retries exactly these")
	}
}

func TestSlowDownOnlyEverIncreasesAndIsCapped(t *testing.T) {
	hl := newHostLimiter(100*time.Millisecond, nil)

	first := hl.slowDown("h", time.Second)
	if first <= 100*time.Millisecond {
		t.Fatalf("a 429 must raise the interval, got %v", first)
	}
	second := hl.slowDown("h", time.Second)
	if second < first {
		t.Fatalf("the interval must never fall: %v then %v", first, second)
	}
	for i := 0; i < 20; i++ {
		hl.slowDown("h", time.Second)
	}
	if got := hl.slowDown("h", time.Second); got > time.Second {
		t.Fatalf("the interval must be capped at max, got %v", got)
	}
}

// TestUndatedLinksSortLastSoTheyCannotRenumberACorpus pins the ordering that makes adding a league-indexed source to an ALREADY-PUBLISHED corpus non-destructive.
func TestUndatedLinksSortLastSoTheyCannotRenumberACorpus(t *testing.T) {
	links := []Link{
		{URL: "https://pobb.in/undated1", PostedAt: ""},
		{URL: "https://pobb.in/dated2", PostedAt: "2026-07-01T00:00:00Z"},
		{URL: "https://pobb.in/undated0", PostedAt: ""},
		{URL: "https://pobb.in/dated1", PostedAt: "2026-02-01T00:00:00Z"},
	}
	sortLinksForNumbering(links)

	want := []string{"dated1", "dated2", "undated0", "undated1"}
	for i, w := range want {
		if got := strings.TrimPrefix(links[i].URL, "https://pobb.in/"); got != w {
			t.Fatalf("position %d = %q, want %q (order: %v)", i, got, w, links)
		}
	}
}

// TestGateTierIsExactlyAcceptedPlusNoPlayerStat pins the criterion ORDER.
func TestGateTierIsExactlyAcceptedPlusNoPlayerStat(t *testing.T) {
	x := string(decodeFixture(t))
	both := removeAllElements(removeAllElements(x, "<Slot "), "<PlayerStat ")

	v := judge([]byte(both), criteria318())
	if v.ok {
		t.Fatal("a build with neither equipment nor PlayerStat was accepted")
	}
	if v.reason != rejUnderEquipped {
		t.Fatalf("a build failing BOTH tiers reported %q; it must report the GATE failure %q, "+
			"or it is counted as gate-eligible when it never was", v.reason, rejUnderEquipped)
	}

	if v := judge([]byte(removeAllElements(x, "<PlayerStat ")), criteria318()); v.reason != rejNoPlayerStat {
		t.Fatalf("a gate-passing build with no PlayerStat reported %q, want %q", v.reason, rejNoPlayerStat)
	}
}

const pbFixture = `<select class="poe-league-select" data-game="poe1">` +
	`<option value="allflame" data-version="3.29" selected>3.29 Allflame</option>` +
	`<option value="mirage" data-version="3.28">3.28 Mirage</option>` +
	`<option value="sanctum" data-version="3.20">3.20 Sanctum</option></select>` +
	`<a href="/poe1/witch/necromancer">x</a><a href="/poe1/witch/occultist">x</a>` +
	`<a href="/poe1/witch/necromancer/minion">x</a>` +
	`<a href="/poe1/author/pohx">x</a>` +
	`<a href="/poe1/witch/necromancer">dup</a>`

func TestPoebuildsSweepsAscendanciesOnly(t *testing.T) {
	got := parsePBSections([]byte(pbFixture))
	want := []string{"/poe1/witch/necromancer", "/poe1/witch/occultist"}
	if len(got) != len(want) {
		t.Fatalf("parsePBSections() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parsePBSections() = %v, want %v", got, want)
		}
	}
}

// TestPoebuildsLeagueTableIsReadFromTheSite pins the version->slug resolution.
func TestPoebuildsLeagueTableIsReadFromTheSite(t *testing.T) {
	ls := parsePBLeagues([]byte(pbFixture))
	if len(ls) != 3 {
		t.Fatalf("parsePBLeagues() = %+v, want 3 rows", ls)
	}
	if ls[0].slug != "allflame" || ls[0].version != "3.29" || !ls[0].selected {
		t.Errorf("row 0 = %+v, want allflame/3.29/selected", ls[0])
	}
	if ls[1].selected || ls[2].selected {
		t.Errorf("only the selected option may carry the flag: %+v", ls)
	}
}

// TestPoebuildsRefusesAThinTaxonomy pins the refusal that keeps a broken parser from looking like a dead source.
func TestPoebuildsRefusesAThinTaxonomy(t *testing.T) {
	if n := len(parsePBSections([]byte(`<a href="/poe1/author/pohx">x</a>`))); n >= minPBSections {
		t.Fatalf("a page with no ascendancy hrefs parsed %d sections — the guard cannot fire", n)
	}
	var full strings.Builder
	for _, class := range []string{"duelist", "marauder", "ranger", "scion", "shadow", "templar", "witch"} {
		for _, asc := range []string{"a", "b", "c"} {
			full.WriteString(`<a href="/poe1/` + class + `/` + asc + `">x</a>`)
		}
	}
	if n := len(parsePBSections([]byte(full.String()))); n < minPBSections {
		t.Fatalf("a full 21-section taxonomy parsed %d — the guard would refuse the live site", n)
	}
}

const paFixture = `<a href="/build/3icirQcy">x</a><a href="/build/3tyhJehe">x</a>` +
	`<a href="/build/3icirQcy?ref=pobarchives">dup with a query string</a>` +
	`<a href="/build/3icirQcy">exact dup</a>` +
	`<a href="/builds/poe?goldenPage=2">2</a><a href="/builds/poe2">poe2</a>`

func TestPobarchivesListingParsesBuildIDsOnce(t *testing.T) {
	got := parsePABuildIDs([]byte(paFixture))
	want := []string{"3icirQcy", "3tyhJehe"}
	if len(got) != len(want) {
		t.Fatalf("parsePABuildIDs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parsePABuildIDs() = %v, want %v", got, want)
		}
	}
	for _, id := range got {
		if strings.HasPrefix(id, "poe") {
			t.Errorf("parsePABuildIDs() picked up a navigation href: %q", id)
		}
	}
}

// TestPobarchivesRefusesAThinListing pins the refusal that separates a broken parser from a genuinely empty tree filter.
func TestPobarchivesRefusesAThinListing(t *testing.T) {
	if n := len(parsePABuildIDs([]byte(`<a href="/builds/poe?goldenPage=2">2</a>`))); n >= minPAListingIDs {
		t.Fatalf("a page with no build hrefs parsed %d ids — the guard cannot fire", n)
	}
	var full strings.Builder
	for i := range 10 {
		fmt.Fprintf(&full, `<a href="/build/id%08d">x</a>`, i)
	}
	if n := len(parsePABuildIDs([]byte(full.String()))); n < minPAListingIDs {
		t.Fatalf("a full 10-row listing page parsed %d — the guard would refuse the live site", n)
	}
}

// TestPoebuildsAuthorSitemapParse pins the AUTHOR arm's enumeration surface.
func TestPoebuildsAuthorSitemapParse(t *testing.T) {
	const sm = `<?xml version="1.0"?><urlset>` +
		`<url><loc>https://www.poebuilds.cc/poe1/author/allie</loc></url>` +
		`<url><loc>https://www.poebuilds.cc/poe1/author/big-guy-grinds</loc></url>` +
		`<url><loc>https://www.poebuilds.cc/poe1/author/allie</loc></url>` +
		`<url><loc>https://www.poebuilds.cc/poe1/witch/necromancer</loc></url>` +
		`<url><loc>https://www.poebuilds.cc/poe2/author/someone</loc></url>` +
		`</urlset>`
	got := parsePBAuthors([]byte(sm))
	want := []string{
		"https://www.poebuilds.cc/poe1/author/allie",
		"https://www.poebuilds.cc/poe1/author/big-guy-grinds",
	}
	if len(got) != len(want) {
		t.Fatalf("parsePBAuthors() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parsePBAuthors() = %v, want %v", got, want)
		}
	}
	if n := len(parsePBAuthors([]byte(sm))); n >= minPBAuthors {
		t.Fatalf("a 2-author sitemap parsed %d — the guard cannot fire", n)
	}
	var full strings.Builder
	for i := range minPBAuthors + 10 {
		fmt.Fprintf(&full, `<url><loc>https://www.poebuilds.cc/poe1/author/a%d</loc></url>`, i)
	}
	if n := len(parsePBAuthors([]byte(full.String()))); n < minPBAuthors {
		t.Fatalf("a full-sized sitemap parsed %d — the guard would refuse the live site", n)
	}
}
