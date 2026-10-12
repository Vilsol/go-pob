package builds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

const richDoc = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a <b> & \"c\"\n"}]}]}`

func TestNotesHashIsSha256OfUTF8(t *testing.T) {
	cases := map[string]string{
		"":                "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"abc":             "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		"^xFF0000red\n^7": "c950ad868af85ca705ab8c9d73d5e41013b2e49c7f2dbf4d48997f8818d339ee",
	}
	for in, want := range cases {
		if got := pob.NotesHash(in); got != want {
			t.Errorf("NotesHash(%q) = %s, want %s", in, got, want)
		}
	}
	if pob.NotesHash("é") == pob.NotesHash("e") {
		t.Error("NotesHash must hash the UTF-8 bytes, not a normalised form")
	}
}

func TestRichNotesStatus(t *testing.T) {
	b := EmptyBuild()
	if got := b.RichNotesStatus(); got != "none" {
		t.Fatalf("no node: %s", got)
	}

	b.SetRichNotes("^1red\nplain", richDoc)
	if got := b.RichNotesStatus(); got != "match" {
		t.Fatalf("fresh write: %s", got)
	}
	if b.Notes != "^1red\nplain" || b.RichNotesDoc() != richDoc {
		t.Fatalf("SetRichNotes did not write both halves: %q / %q", b.Notes, b.RichNotesDoc())
	}

	b.SetNotes("^1red\nplain edited in PoB")
	if got := b.RichNotesStatus(); got != "mismatch" {
		t.Fatalf("plain edited outside the rich editor: %s", got)
	}

	b.SetRichNotes("x", richDoc)
	b.RichNotes.V = pob.RichNotesVersion + 1
	if got := b.RichNotesStatus(); got != "none" {
		t.Fatalf("unknown version must not be loaded: %s", got)
	}
}

// TestRichNotesRoundTripsThroughXML: A build XML with the node round-trips it, and <Notes> loses nothing on the way.
func TestRichNotesRoundTripsThroughXML(t *testing.T) {
	b := EmptyBuild()
	plain := "^xAF6025Unique^7 line\n\n\ttabbed ^1 < & > \" '"
	b.SetRichNotes(plain, richDoc)

	raw, err := SerializeBuild(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`<RichNotes format="tiptap-json" v="1" plainHash="`+pob.NotesHash(plain)+`">`)) {
		t.Fatalf("node not written as specified:\n%s", raw)
	}

	again, err := ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	if again.Notes != plain {
		t.Fatalf("<Notes> changed: %q", again.Notes)
	}
	if !reflect.DeepEqual(again.RichNotes, b.RichNotes) {
		t.Fatalf("<RichNotes> changed: %+v vs %+v", again.RichNotes, b.RichNotes)
	}
	if again.RichNotesStatus() != "match" {
		t.Fatalf("status after reload: %s", again.RichNotesStatus())
	}
}

// TestRichNotesCorpusRoundTrip: Over real corpus builds carrying notes.
func TestRichNotesCorpusRoundTrip(t *testing.T) {
	dir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	withNotes := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(src, []byte("<Notes>")) {
			continue
		}

		plainBuild, err := ParseBuild(src)
		if err != nil {
			continue
		}
		if strings.TrimSpace(plainBuild.Notes) == "" {
			continue
		}
		withNotes++

		plainRaw, err := SerializeBuild(plainBuild)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(plainRaw, []byte("RichNotes")) {
			t.Fatalf("%s: a build without the node serialised one", e.Name())
		}

		richBuild, _ := ParseBuild(src)
		richBuild.SetRichNotes(richBuild.Notes, richDoc)
		richRaw, err := SerializeBuild(richBuild)
		if err != nil {
			t.Fatal(err)
		}

		fromPlain, err := ParseBuild(plainRaw)
		if err != nil {
			t.Fatal(err)
		}
		fromRich, err := ParseBuild(richRaw)
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		if fromRich.Notes != plainBuild.Notes {
			t.Fatalf("%s: <Notes> changed through the rich round trip", e.Name())
		}
		if fromRich.RichNotesStatus() != "match" {
			t.Fatalf("%s: status %s after round trip", e.Name(), fromRich.RichNotesStatus())
		}
		if !reflect.DeepEqual(fromRich.RichNotes, richBuild.RichNotes) {
			t.Fatalf("%s: <RichNotes> changed", e.Name())
		}
		fromRich.RichNotes = nil
		if !reflect.DeepEqual(fromRich, fromPlain) {
			t.Fatalf("%s: the node changed another part of the parse", e.Name())
		}
	}
	if withNotes == 0 {
		t.Fatal("no corpus build carried notes: the loop proved nothing")
	}
	t.Logf("%d corpus builds with notes round-tripped", withNotes)
}

const (
	boundaryLabel    = "Main \"6L\" it's\nline two\ttabbed"
	boundaryBuff     = "+10% to \"Fire\" Resistance\nit's a second line\tafter a tab"
	boundaryModTitle = "Say \"hi\" it's"
	boundaryModText  = "+10 to \"Strength\"\n+5 to maximum Life"
)

func withBoundaryText(b *pob.PathOfBuilding) {
	b.Skills.SkillSets[0].Skills[0].Label = boundaryLabel
	kept := b.Party.ImportedBuffs[:0]
	for _, buff := range b.Party.ImportedBuffs {
		if buff.Name != "EnemyMods" {
			kept = append(kept, buff)
		}
	}
	b.Party.ImportedBuffs = append(kept, pob.PartyBuffs{Name: "EnemyMods", Text: boundaryBuff})
	title := boundaryModTitle
	enabled := "true"
	b.Config.CustomModifierBlocks = append(b.Config.CustomModifierBlocks, pob.CustomModifierBlock{Title: &title, Enabled: &enabled, Text: boundaryModText})
}

func TestSerializeBuildBoundaryTextRoundTripsInGo(t *testing.T) {
	b := EmptyBuild()
	withBoundaryText(b)
	raw, err := SerializeBuild(b)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Skills.SkillSets[0].Skills[0].Label; got != boundaryLabel {
		t.Errorf("attribute: %q", got)
	}
	if got := again.Party.ImportedBuffs; len(got) != 1 || got[0].Text != boundaryBuff {
		t.Errorf("party text: %+v", got)
	}
	if got := again.Config.CustomModifierBlocks; len(got) != 1 || *got[0].Title != boundaryModTitle || got[0].Text != boundaryModText {
		t.Errorf("custom mod block: %+v", got)
	}
}

// TestSerializeBuildPoBEntitiesAreInvisibleToGo: SerializeBuild rewrites encoding/xml's numeric references into forms PoB's reader decodes.
func TestSerializeBuildPoBEntitiesAreInvisibleToGo(t *testing.T) {
	dir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	rewritten := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		b, err := ParseBuild(src)
		if err != nil {
			continue
		}
		std, err := xml.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		ours, err := SerializeBuild(b)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(std, ours) {
			continue
		}
		rewritten++
		for _, ent := range []string{"&#xA;", "&#x9;", "&#34;", "&#39;"} {
			if bytes.Contains(ours, []byte(ent)) {
				t.Fatalf("%s: %s survived SerializeBuild", e.Name(), ent)
			}
		}

		fromStd, err := ParseBuild(std)
		if err != nil {
			t.Fatal(err)
		}
		fromOurs, err := ParseBuild(ours)
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		if !reflect.DeepEqual(fromStd, fromOurs) {
			t.Fatalf("%s: Go reads SerializeBuild's output differently from encoding/xml's", e.Name())
		}
	}
	if rewritten == 0 {
		t.Fatal("no corpus build needed a rewrite: the loop proved nothing")
	}
	t.Logf("%d corpus builds rewritten and read back identically", rewritten)
}

const (
	pobLoadSource  = "0758-necromancer-vaal-summon-skeletons-of-archers.xml"
	pobLoadFixture = "../testdata/notes/rich-notes-pob-load.xml"
	pobLoadOracle  = "../testdata/oracle/rich_notes_pob_load.json"
	pobLoadExtra   = "\n\"quoted\" it's\ttabbed <tag> & ^1red"
	hostileDoc     = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"</RichNotes><Notes>x</Notes> ]]> &amp; ^1 \t"}]}]}`
)

// TestRichNotesPoBLoadFixtureIsCurrent: The XML the pinned PoB is fed by oracle/rich_notes_pob_load.lua (task rich-notes-pob).
func TestRichNotesPoBLoadFixtureIsCurrent(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), pobLoadSource))
	if err != nil {
		t.Skipf("corpus build unavailable: %v", err)
	}
	b, err := ParseBuild(src)
	if err != nil {
		t.Fatal(err)
	}
	b.SetRichNotes(b.Notes+pobLoadExtra, hostileDoc)
	withBoundaryText(b)
	raw, err := SerializeBuild(b)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("RICH_NOTES_WRITE_FIXTURE") == "1" {
		if err := os.MkdirAll(filepath.Dir(pobLoadFixture), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pobLoadFixture, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	committed, err := os.ReadFile(pobLoadFixture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(committed, raw) {
		t.Fatalf("%s is stale against the Go writer: rerun with RICH_NOTES_WRITE_FIXTURE=1, then task rich-notes-pob", pobLoadFixture)
	}
}

// TestRichNotesLoadInPinnedPoB: What the pinned PoB read out of the fixture.
func TestRichNotesLoadInPinnedPoB(t *testing.T) {
	raw, err := os.ReadFile(pobLoadFixture)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	if b.RichNotesStatus() != "match" {
		t.Fatalf("fixture carries no matching rich node: %s", b.RichNotesStatus())
	}

	oracleRaw, err := os.ReadFile(pobLoadOracle)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Notes             string `json:"notes"`
		SavedHasRichNotes bool   `json:"savedHasRichNotes"`
		SavedNotes        string `json:"savedNotes"`

		FirstSocketGroupLabel string `json:"firstSocketGroupLabel"`
		EnemyMods             string `json:"enemyMods"`
		CustomMods            []struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		} `json:"customMods"`
	}
	if err := json.Unmarshal(oracleRaw, &got); err != nil {
		t.Fatal(err)
	}

	if got.Notes != b.Notes {
		t.Fatalf("PoB's NotesTab holds a different text than the <Notes> go-pob wrote:\nPoB: %q\nGo:  %q", got.Notes, b.Notes)
	}
	if !strings.Contains(got.Notes, "^xE05030No guard") || !strings.Contains(got.Notes, pobLoadExtra) {
		t.Fatalf("the probe read an unrelated text: %q", got.Notes)
	}
	if got.SavedHasRichNotes {
		t.Fatal("PoB's SaveDB kept <RichNotes>; the mismatch design assumes it drops it")
	}
	if got.SavedNotes != b.Notes {
		t.Fatalf("PoB's re-save changed <Notes>:\nPoB: %q\nGo:  %q", got.SavedNotes, b.Notes)
	}

	if got.FirstSocketGroupLabel != boundaryLabel {
		t.Errorf("attribute value: PoB read %q, go-pob wrote %q", got.FirstSocketGroupLabel, boundaryLabel)
	}
	if got.EnemyMods != boundaryBuff {
		t.Errorf("party text: PoB read %q, go-pob wrote %q", got.EnemyMods, boundaryBuff)
	}
	found := false
	for _, m := range got.CustomMods {
		if m.Title == boundaryModTitle {
			found = true
			if m.Text != boundaryModText {
				t.Errorf("custom mod text: PoB read %q, go-pob wrote %q", m.Text, boundaryModText)
			}
		}
	}
	if !found {
		t.Errorf("custom mod block titled %q not read by PoB: %+v", boundaryModTitle, got.CustomMods)
	}
}

// TestNotesParseTrimsLikePoB: PoB's reader trims element text.
func TestNotesParseTrimsLikePoB(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), pobLoadSource))
	if err != nil {
		t.Skipf("corpus build unavailable: %v", err)
	}
	if !bytes.Contains(src, []byte("<Notes>\n")) {
		t.Fatal("the source build's notes no longer start with whitespace: this test proves nothing")
	}
	b, err := ParseBuild(src)
	if err != nil {
		t.Fatal(err)
	}

	oracleRaw, err := os.ReadFile(pobLoadOracle)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Notes string `json:"notes"`
	}
	if err := json.Unmarshal(oracleRaw, &got); err != nil {
		t.Fatal(err)
	}
	if want := strings.TrimSuffix(got.Notes, pobLoadExtra); b.Notes != want {
		t.Fatalf("Go's <Notes> differs from PoB's:\nGo:  %q\nPoB: %q", b.Notes, want)
	}
}
