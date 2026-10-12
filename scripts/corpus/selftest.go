package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdSelftest(args []string) error {
	fs := flag.NewFlagSet("selftest", flag.ExitOnError)
	corpus := fs.String("corpus", "", "optional: an existing corpus dir to use as the live control (e.g. testdata/corpus-3.29)")
	corpusTree := fs.String("corpus-tree", "3_29", "tree version that control corpus is pinned to")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	banner("selftest: every check must be able to return both answers")

	var failures []string
	check := func(name string, got, want bool) {
		state := "ok  "
		if got != want {
			state = "FAIL"
			failures = append(failures, name)
		}
		fmt.Printf("  [%s] %s\n", state, name)
	}

	c := criteria{treeVersion: "3_19", minNodes: 80, minSlots: 8, slotMode: "occurrences"}

	check("accepts a build whose active spec is 3_19",
		judge(synth(synthOpts{}), c).ok, true)
	check("rejects a build whose active spec is 3_18",
		judge(synth(synthOpts{tree: "3_18"}), c).reason == rejWrongTree, true)
	check("reads the ACTIVE spec, not the first (3_18 first, 3_19 active)",
		judge(synth(synthOpts{extraSpecFirst: "3_18", activeSpec: 2}), c).ok, true)
	check("reads the ACTIVE spec, not the first (3_19 first, 3_18 active)",
		judge(synth(synthOpts{tree: "3_18", extraSpecFirst: "3_19", activeSpec: 2}), c).reason == rejWrongTree, true)

	check("rejects a build with no <PlayerStat> list",
		judge(synth(synthOpts{noStats: true}), c).reason == rejNoPlayerStat, true)

	check("rejects a stub tree (79 nodes)",
		judge(synth(synthOpts{nodes: 79}), c).reason == rejStubTree, true)
	check("accepts exactly at the node threshold (80)",
		judge(synth(synthOpts{nodes: 80}), c).ok, true)
	check("rejects an under-equipped character (7 slots)",
		judge(synth(synthOpts{slots: 7}), c).reason == rejUnderEquipped, true)
	check("accepts exactly at the slot threshold (8)",
		judge(synth(synthOpts{slots: 8}), c).ok, true)
	check("does not count flasks toward equipment slots",
		judge(synth(synthOpts{slots: 7, flasks: 5}), c).reason == rejUnderEquipped, true)

	check("rejects non-XML as xml_parse_failed",
		judge([]byte("this is not xml at all"), c).reason == rejXMLParseFailed, true)
	check("rejects XML with no <Spec>",
		judge([]byte(`<PathOfBuilding><Build/><Tree activeSpec="1"/></PathOfBuilding>`), c).reason == rejNoTree, true)

	xmlDoc := synth(synthOpts{})
	code := encodeExport(xmlDoc)
	rt, err := decodeExport([]byte(code))
	check("decodeExport round-trips a code it just encoded", err == nil && bytes.Equal(rt, xmlDoc), true)
	_, err = decodeExport([]byte(strings.Repeat("A", 400)))
	check("decodeExport fails on base64 that is not a zlib stream", err != nil, true)
	check("decodeExport tolerates the newlines pastebin inserts",
		mustRoundTrip(insertNewlines(code, 60), xmlDoc), true)

	pobbPage := `<html><body><textarea class="flex-auto" aria-label="Path of Building buildcode" readonly="">` +
		code + `</textarea></body></html>`
	got, ok := extractEmbeddedCode([]byte(pobbPage))
	check("extracts a code from a pobb.in <textarea>", ok && mustRoundTrip(string(got), xmlDoc), true)

	pastebinPage := `<html><body><ol class="text"><li class="li1"><div class="de1">` + code +
		`</div></li></ol></body></html>`
	got, ok = extractEmbeddedCode([]byte(pastebinPage))
	check("extracts a code from a pastebin <ol class=text>", ok && mustRoundTrip(string(got), xmlDoc), true)

	decoy := `<html><script src="data:application/javascript;base64,` + strings.Repeat("QUJD", 200) +
		`"></script><textarea aria-label="Path of Building buildcode">` + code + `</textarea></html>`
	got, ok = extractEmbeddedCode([]byte(decoy))
	check("ignores a long base64 decoy and returns the real code", ok && mustRoundTrip(string(got), xmlDoc), true)

	_, ok = extractEmbeddedCode([]byte(`<html><body><h1>This paste has been removed.</h1></body></html>`))
	check("returns NO code from a page that carries none", ok, false)

	notPoB := encodeExport([]byte("<SomethingElse>not a build</SomethingElse>"))
	_, ok = extractEmbeddedCode([]byte(`<html><textarea>` + notPoB + `</textarea></html>`))
	check("refuses a valid zlib blob that is not a PoB document", ok, false)

	check("classifies a Cloudflare interstitial as BLOCKED",
		classifyBody([]byte(`<!DOCTYPE html><html><head><title>Just a moment...</title>`)) == fetchBlocked, true)
	check("classifies unexpected HTML on a PAYLOAD endpoint as BLOCKED",
		classifyBody([]byte(`<html><body>some page we have never seen</body></html>`)) == fetchBlocked, true)
	_, b1 := looksBlocked([]byte(`<!doctype html><html><body>forum listing</body></html>`), true)
	check("does NOT call ordinary HTML blocked where HTML is expected", b1, false)
	_, b2 := looksBlocked([]byte(`<!doctype html><html><title>Just a moment...</title>`), true)
	check("still catches a named challenge page where HTML is expected", b2, true)
	check("classifies a real export code as OK",
		classifyBody([]byte(code)) == fetchOK, true)
	check("classifies a short plaintext body as NOTCODE, not blocked",
		classifyBody([]byte("Error, this paste has been removed.")) == fetchNotCode, true)
	check("blocked and transport are RETRYABLE",
		!fetchBlocked.terminal() && !fetchTransport.terminal(), true)
	check("ok and notfound are TERMINAL",
		fetchOK.terminal() && fetchNotFound.terminal(), true)

	links, _ := extractLinks("see https://pastebin.com/aBcD1234 and https://pobb.in/5v25ibooPzOx here")
	check("extracts a well-formed pastebin id and pobb.in id", len(links) == 2, true)
	_, rejected := extractLinks("https://pastebin.com/u/someuser profile page")
	check("REJECTS an over-matched pastebin id rather than emitting a dead link", rejected["pastebin.com"] > 0, true)
	links, _ = extractLinks("wrapped in markdown [pob](https://pastebin.com/aBcD1234).")
	check("survives trailing punctuation", len(links) == 1 && links[0].ID == "aBcD1234", true)
	links, _ = extractLinks("raw form https://pastebin.com/raw/aBcD1234")
	check("normalises a /raw/ link to the same id", len(links) == 1 && links[0].ID == "aBcD1234", true)
	links, _ = extractLinks("https://pobb.in/u/alice/5v25ibooPzOx")
	check("parses the user-scoped pobb.in shape rather than capturing \"u\"",
		len(links) == 1 && links[0].ID == "5v25ibooPzOx", true)
	links, rejected = extractLinks("profile only https://pobb.in/u/alice")
	check("still REJECTS a bare pobb.in profile URL, which carries no paste",
		len(links) == 0 && rejected["pobb.in"] > 0, true)
	links, _ = extractLinks(`https://pobb.in/eVgBOQGYn\_JH`)
	check("unescapes a markdown-escaped underscore instead of truncating the id",
		len(links) == 1 && links[0].ID == "eVgBOQGYn_JH", true)
	links, _ = extractLinks(`"description":"https://pobb.in/eVgBOQGYn\\_JH\n"`)
	check("stops at a JSON CONTROL escape rather than absorbing it into the id",
		len(links) == 1 && links[0].ID == "eVgBOQGYn_JH", true)

	d, _ := parsePoB(synth(synthOpts{}))
	check("names a gem-driven build after its first non-support gem", d.mainSkillLabel() == "boneshatter", true)
	d, _ = parsePoB(synth(synthOpts{itemGranted: true}))
	check("names an item-granted skill group -itemskill-", strings.HasPrefix(d.mainSkillLabel(), "itemskill-"), true)

	if *corpus != "" {
		fmt.Println()
		cc := criteria{treeVersion: *corpusTree, minNodes: 80, minSlots: 8, slotMode: "occurrences"}
		err := validateExistingCorpus(*corpus, cc, false)
		check(fmt.Sprintf("live control: every build in %s passes unchanged", *corpus), err == nil, true)
		if err != nil {
			fmt.Printf("       %v\n", err)
		}
	} else {
		fmt.Println("\n  note: -corpus not given, so the LIVE control did not run. A synthetic pass alone")
		fmt.Println("        does not show these criteria reproduce the shipped corpus.")
	}

	fmt.Println()
	if len(failures) > 0 {
		return fmt.Errorf("%d selftest checks failed: %s", len(failures), strings.Join(failures, "; "))
	}
	fmt.Println("all selftest checks passed — every filter demonstrated both outcomes")
	return nil
}

func classifyBody(body []byte) fetchClass {
	if _, blocked := looksBlocked(body, false); blocked {
		return fetchBlocked
	}
	if !looksLikeExportCode(body) {
		return fetchNotCode
	}
	return fetchOK
}

func encodeExport(x []byte) string {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, _ = zw.Write(x)
	_ = zw.Close()
	return base64.URLEncoding.EncodeToString(buf.Bytes())
}

func mustRoundTrip(code string, want []byte) bool {
	got, err := decodeExport([]byte(code))
	return err == nil && bytes.Equal(got, want)
}

func insertNewlines(s string, every int) string {
	var b strings.Builder
	for i := 0; i < len(s); i += every {
		b.WriteString(s[i:min(i+every, len(s))])
		b.WriteByte('\n')
	}
	return b.String()
}

type synthOpts struct {
	tree           string
	extraSpecFirst string
	activeSpec     int
	nodes          int
	slots          int
	flasks         int
	noStats        bool
	itemGranted    bool
}

func synth(o synthOpts) []byte {
	if o.tree == "" {
		o.tree = "3_19"
	}
	if o.activeSpec == 0 {
		o.activeSpec = 1
	}
	if o.nodes == 0 {
		o.nodes = 120
	}
	if o.slots == 0 {
		o.slots = 10
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n<PathOfBuilding>\n")
	b.WriteString(`<Build level="95" className="Marauder" ascendClassName="Juggernaut" mainSocketGroup="1">` + "\n")
	if !o.noStats {
		b.WriteString(`<PlayerStat stat="TotalDPS" value="123456"/>` + "\n")
	}
	b.WriteString("</Build>\n")

	fmt.Fprintf(&b, `<Tree activeSpec="%d">`+"\n", o.activeSpec)
	if o.extraSpecFirst != "" {
		fmt.Fprintf(&b, `<Spec treeVersion="%s" classId="1" nodes="%s"/>`+"\n", o.extraSpecFirst, nodeList(120))
	}
	fmt.Fprintf(&b, `<Spec treeVersion="%s" classId="1" nodes="%s"/>`+"\n", o.tree, nodeList(o.nodes))
	b.WriteString("</Tree>\n")

	b.WriteString("<Items activeItemSet=\"1\">\n<ItemSet>\n")
	names := []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt"}
	for i := 0; i < o.slots && i < len(names); i++ {
		fmt.Fprintf(&b, `<Slot name="%s" itemId="%d"/>`+"\n", names[i], i+1)
	}
	for i := 0; i < o.flasks; i++ {
		fmt.Fprintf(&b, `<Slot name="Flask %d" itemId="%d"/>`+"\n", i+1, 100+i)
	}
	b.WriteString("</ItemSet>\n</Items>\n")

	b.WriteString("<Skills>\n")
	if o.itemGranted {
		b.WriteString(`<Skill source="Item:4:Soulwrest, Ezomyte Staff" enabled="true"></Skill>` + "\n")
	} else {
		b.WriteString(`<Skill enabled="true">` + "\n")
		b.WriteString(`<Gem nameSpec="Boneshatter" skillId="Boneshatter" enabled="true"/>` + "\n")
		b.WriteString(`<Gem nameSpec="Melee Physical Damage" skillId="SupportMeleePhysicalDamage" enabled="true"/>` + "\n")
		b.WriteString("</Skill>\n")
	}
	b.WriteString("</Skills>\n</PathOfBuilding>\n")
	return []byte(b.String())
}

func nodeList(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprint(1000 + i)
	}
	return strings.Join(parts, ",")
}

func resolveCorpus(p string) string {
	if p == "" {
		return p
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return filepath.Clean(p)
}
