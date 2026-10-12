package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1" //nolint:gosec // G505: content identity, not security — see sha1hex
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var reNotBase64 = regexp.MustCompile(`[^A-Za-z0-9+/=_-]`)

var errEmptyCode = errors.New("empty code")

func looksLikeExportCode(b []byte) bool {
	s := strings.TrimSpace(string(b))
	if len(s) < 200 {
		return false
	}
	return len(reNotBase64.ReplaceAllString(s, "")) >= len(s)*98/100
}

func decodeExport(code []byte) ([]byte, error) {
	s := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, string(code))
	if s == "" {
		return nil, fmt.Errorf("decodeExport: %w", errEmptyCode)
	}

	encodings := []*base64.Encoding{
		base64.URLEncoding, base64.RawURLEncoding,
		base64.StdEncoding, base64.RawStdEncoding,
	}
	var raw []byte
	var lastErr error
	for _, enc := range encodings {
		d, err := enc.DecodeString(s)
		if err == nil {
			raw = d
			break
		}
		lastErr = err
	}
	if raw == nil {
		return nil, fmt.Errorf("base64: %w", lastErr)
	}

	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("zlib: %w", err)
	}
	defer zr.Close()

	xmlBytes, err := io.ReadAll(io.LimitReader(zr, 64<<20))
	if err != nil && len(xmlBytes) == 0 {
		return nil, fmt.Errorf("inflate: %w", err)
	}
	return xmlBytes, nil
}

//nolint:gosec // G401: identity hash, see above
func sha1hex(b []byte) string { return fmt.Sprintf("%x", sha1.Sum(b)) }

var (
	reHTMLTag    = regexp.MustCompile(`(?s)<[^>]*>`)
	reBase64Run  = regexp.MustCompile(`[A-Za-z0-9+/=_-]{200,}`)
	reHTMLEntity = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'")
)

func extractEmbeddedCode(page []byte) ([]byte, bool) {
	if looksLikeExportCode(page) {
		if x, err := decodeExport(page); err == nil && isPoBXML(x) {
			return bytes.TrimSpace(page), true
		}
	}

	text := reHTMLEntity.Replace(reHTMLTag.ReplaceAllString(string(page), "\n"))
	for _, cand := range reBase64Run.FindAllString(text, -1) {
		x, err := decodeExport([]byte(cand))
		if err == nil && isPoBXML(x) {
			return []byte(cand), true
		}
	}
	return nil, false
}

func isPoBXML(x []byte) bool {
	head := x[:min(len(x), 4096)]
	return bytes.Contains(head, []byte("<PathOfBuilding"))
}

type pobSlot struct {
	Name   string `xml:"name,attr"`
	ItemID string `xml:"itemId,attr"`
}

type pobGem struct {
	NameSpec string `xml:"nameSpec,attr"`
	SkillID  string `xml:"skillId,attr"`
	Enabled  string `xml:"enabled,attr"`
}

type pobSkill struct {
	Source  string   `xml:"source,attr"`
	Enabled string   `xml:"enabled,attr"`
	Gems    []pobGem `xml:"Gem"`
}

type pobDoc struct {
	XMLName xml.Name `xml:"PathOfBuilding"`
	Build   struct {
		Level           int    `xml:"level,attr"`
		ClassName       string `xml:"className,attr"`
		AscendClassName string `xml:"ascendClassName,attr"`
		MainSocketGroup int    `xml:"mainSocketGroup,attr"`
		PlayerStats     []struct {
			Stat  string `xml:"stat,attr"`
			Value string `xml:"value,attr"`
		} `xml:"PlayerStat"`
	} `xml:"Build"`
	Tree struct {
		ActiveSpec int `xml:"activeSpec,attr"`
		Specs      []struct {
			TreeVersion string `xml:"treeVersion,attr"`
			Nodes       string `xml:"nodes,attr"`
			ClassID     string `xml:"classId,attr"`
		} `xml:"Spec"`
	} `xml:"Tree"`
	Items struct {
		Slots    []pobSlot `xml:"Slot"`
		ItemSets []struct {
			Slots []pobSlot `xml:"Slot"`
		} `xml:"ItemSet"`
	} `xml:"Items"`
	Skills struct {
		SkillSets []struct {
			Skills []pobSkill `xml:"Skill"`
		} `xml:"SkillSet"`
		Skills []pobSkill `xml:"Skill"`
	} `xml:"Skills"`
}

func parsePoB(x []byte) (*pobDoc, error) {
	var d pobDoc
	dec := xml.NewDecoder(bytes.NewReader(x))
	dec.Strict = false
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("parsing PoB XML: %w", err)
	}
	return &d, nil
}

var equipmentSlots = map[string]bool{
	"Weapon 1": true, "Weapon 2": true, "Helmet": true, "Body Armour": true,
	"Gloves": true, "Boots": true, "Amulet": true, "Ring 1": true, "Ring 2": true,
	"Belt": true,
}

func (d *pobDoc) activeSpecIndex() int {
	i := d.Tree.ActiveSpec
	if i < 1 || i > len(d.Tree.Specs) {
		return 1
	}
	return i
}

func (d *pobDoc) activeTreeVersion() string {
	i := d.activeSpecIndex()
	if i > len(d.Tree.Specs) {
		return ""
	}
	return d.Tree.Specs[i-1].TreeVersion
}

func (d *pobDoc) allocatedNodes() int {
	i := d.activeSpecIndex()
	if i > len(d.Tree.Specs) {
		return 0
	}
	n := 0
	for _, part := range strings.Split(d.Tree.Specs[i-1].Nodes, ",") {
		if strings.TrimSpace(part) != "" {
			n++
		}
	}
	return n
}

func (d *pobDoc) filledEquipmentSlots(mode string) int {
	count := func(slots []pobSlot, into map[string]bool) int {
		n := 0
		for _, s := range slots {
			if !equipmentSlots[s.Name] || s.ItemID == "" || s.ItemID == "0" {
				continue
			}
			if into != nil {
				into[s.Name] = true
				continue
			}
			n++
		}
		return n
	}

	switch mode {
	case "distinct":
		seen := map[string]bool{}
		count(d.Items.Slots, seen)
		for _, set := range d.Items.ItemSets {
			count(set.Slots, seen)
		}
		return len(seen)
	default:
		n := count(d.Items.Slots, nil)
		for _, set := range d.Items.ItemSets {
			n += count(set.Slots, nil)
		}
		return n
	}
}

func (d *pobDoc) playerStats() int { return len(d.Build.PlayerStats) }

func (d *pobDoc) allSkills() []pobSkill {
	if len(d.Skills.Skills) > 0 {
		return d.Skills.Skills
	}
	var out []pobSkill
	for _, ss := range d.Skills.SkillSets {
		out = append(out, ss.Skills...)
	}
	return out
}

var reItemSource = regexp.MustCompile(`^Item:\d+:([^,]+)`)

func (d *pobDoc) mainSkillLabel() string {
	skills := d.allSkills()
	i := d.Build.MainSocketGroup
	if i < 1 || i > len(skills) {
		if len(skills) == 0 {
			return "unknown"
		}
		i = 1
	}
	sk := skills[i-1]

	for _, g := range sk.Gems {
		if g.NameSpec == "" || strings.HasPrefix(g.SkillID, "Support") {
			continue
		}
		return slug(g.NameSpec)
	}

	switch {
	case strings.HasPrefix(sk.Source, "Item:"):
		if m := reItemSource.FindStringSubmatch(sk.Source); m != nil {
			return "itemskill-" + slug(m[1])
		}
		return "itemskill"
	case strings.HasPrefix(sk.Source, "Tree:"):
		return "treeskill"
	case sk.Source != "":
		return "treeskill"
	}
	return "unknown"
}

func (d *pobDoc) classLabel() string {
	if a := strings.TrimSpace(d.Build.AscendClassName); a != "" && a != "None" {
		return slug(a)
	}
	if c := strings.TrimSpace(d.Build.ClassName); c != "" {
		return slug(c) + "-none"
	}
	return "none"
}

var reSlugStrip = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "&apos;", "")
	s = strings.ReplaceAll(s, "'", "")
	s = reSlugStrip.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

const (
	rejWrongTree      = "wrong_active_tree"
	rejNoTree         = "no_tree_spec"
	rejNoPlayerStat   = "no_playerstat"
	rejUnderEquipped  = "under_equipped"
	rejStubTree       = "stub_tree"
	rejDecodeFailed   = "decode_failed"
	rejNotCode        = "not_an_export_code"
	rejXMLParseFailed = "xml_parse_failed"
	rejDuplicate      = "duplicate_content"
	rejUnreachable    = "unreachable"
)

type criteria struct {
	treeVersion string
	minNodes    int
	minSlots    int
	slotMode    string
}

type verdict struct {
	ok       bool
	reason   string
	detail   string
	doc      *pobDoc
	xmlSha1  string
	nodes    int
	slots    int
	stats    int
	treeSeen string
}

func judge(x []byte, c criteria) verdict {
	doc, err := parsePoB(x)
	if err != nil {
		return verdict{reason: rejXMLParseFailed, detail: err.Error()}
	}

	v := verdict{
		doc:      doc,
		xmlSha1:  sha1hex(x),
		nodes:    doc.allocatedNodes(),
		slots:    doc.filledEquipmentSlots(c.slotMode),
		stats:    doc.playerStats(),
		treeSeen: doc.activeTreeVersion(),
	}

	if len(doc.Tree.Specs) == 0 {
		v.reason = rejNoTree
		return v
	}
	if v.treeSeen != c.treeVersion {
		v.reason = rejWrongTree
		v.detail = v.treeSeen
		return v
	}
	if v.nodes < c.minNodes {
		v.reason = rejStubTree
		v.detail = strconv.Itoa(v.nodes) + " nodes"
		return v
	}
	if v.slots < c.minSlots {
		v.reason = rejUnderEquipped
		v.detail = strconv.Itoa(v.slots) + " slots"
		return v
	}
	if v.stats == 0 {
		v.reason = rejNoPlayerStat
		return v
	}

	v.ok = true
	return v
}
