package builds

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/Vilsol/go-pob/pob"
)

type xmlSection struct {
	name       string
	start, end int
}

func splitSections(doc []byte) ([]byte, string, []xmlSection, bool) {
	var prefix []byte
	var root string
	var sections []xmlSection
	depth := 0
	pos := 0
	for {
		i := bytes.IndexByte(doc[pos:], '<')
		if i < 0 {
			return prefix, root, sections, depth == 0 && root != ""
		}
		at := pos + i
		rest := doc[at:]
		switch {
		case bytes.HasPrefix(rest, []byte("<?")):
			end := bytes.Index(rest, []byte("?>"))
			if end < 0 {
				return nil, "", nil, false
			}
			pos = at + end + 2
			continue
		case bytes.HasPrefix(rest, []byte("<!--")):
			end := bytes.Index(rest[4:], []byte("-->"))
			if end < 0 {
				return nil, "", nil, false
			}
			pos = at + 4 + end + 3
			continue
		case bytes.HasPrefix(rest, []byte("<![CDATA[")):
			end := bytes.Index(rest, []byte("]]>"))
			if end < 0 || depth == 0 {
				return nil, "", nil, false
			}
			pos = at + end + 3
			continue
		case bytes.HasPrefix(rest, []byte("<!")):
			return nil, "", nil, false
		}

		closing := len(rest) > 1 && rest[1] == '/'
		end := tagEnd(rest)
		if end < 0 {
			return nil, "", nil, false
		}
		tagClose := at + end + 1
		nameStart := 1
		if closing {
			nameStart = 2
		}
		nameEnd := nameStart
		for nameEnd < end && !isTagSpace(rest[nameEnd]) && rest[nameEnd] != '/' && rest[nameEnd] != '>' {
			nameEnd++
		}
		name := string(rest[nameStart:nameEnd])
		selfClosing := !closing && rest[end-1] == '/'

		switch {
		case closing:
			depth--
			switch depth {
			case 1:
				sections[len(sections)-1].end = tagClose
			case 0:
				if name != root {
					return nil, "", nil, false
				}
			}
			if depth < 0 {
				return nil, "", nil, false
			}
		case depth == 0:
			if root != "" || selfClosing {
				return nil, "", nil, false
			}
			root = name
			prefix = doc[:tagClose]
			depth = 1
		default:
			if depth == 1 {
				sections = append(sections, xmlSection{name: name, start: at, end: -1})
				if selfClosing {
					sections[len(sections)-1].end = tagClose
				}
			}
			if !selfClosing {
				depth++
			}
		}
		pos = tagClose
	}
}

func tagEnd(rest []byte) int {
	var quote byte
	for i := 1; i < len(rest); i++ {
		c := rest[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			return i
		}
	}
	return -1
}

func isTagSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

var reusedSections = map[string]bool{"Items": true, "Tree": true}

func ParseBuildOver(live *pob.PathOfBuilding, liveXML string, rawXML string) (*pob.PathOfBuilding, error) {
	if live == nil {
		return ParseBuildStr(rawXML)
	}
	doc := []byte(rawXML)
	prefix, root, sections, ok := splitSections(doc)
	livePrefix, liveRoot, liveSections, liveOK := splitSections([]byte(liveXML))
	if !ok || !liveOK || root != liveRoot || !bytes.Equal(prefix, livePrefix) || len(sections) != len(liveSections) {
		return ParseBuild(doc)
	}

	liveByName := make(map[string]string, len(liveSections))
	for _, s := range liveSections {
		if _, dup := liveByName[s.name]; dup {
			return ParseBuild(doc)
		}
		liveByName[s.name] = liveXML[s.start:s.end]
	}

	var changed bytes.Buffer
	changed.Write(prefix)
	fresh := make(map[string]bool, len(sections))
	for _, s := range sections {
		before, found := liveByName[s.name]
		if !found || fresh[s.name] {
			return ParseBuild(doc)
		}
		if !reusedSections[s.name] || before != rawXML[s.start:s.end] {
			changed.Write(doc[s.start:s.end])
			fresh[s.name] = true
		}
		delete(liveByName, s.name)
	}
	changed.WriteString("</" + root + ">")

	var parsed pob.PathOfBuilding
	parsed.TreeView.ShowStatDifferences = true
	if err := xml.Unmarshal(stripNilAttrs(changed.Bytes()), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse build as xml: %w", err)
	}

	build := *live
	for name := range fresh {
		switch name {
		case "Build":
			build.Build = parsed.Build
		case "Tree":
			build.Tree = parsed.Tree
		case "Calcs":
			build.Calcs = parsed.Calcs
		case "Notes":
			build.Notes = strings.Trim(parsed.Notes, pob.NotesEdgeSpace)
		case "Items":
			build.Items = parsed.Items
		case "Skills":
			finishSkills(&parsed)
			build.Skills = parsed.Skills
		case "TreeView":
			build.TreeView = parsed.TreeView
		case "Config":
			resolveActiveConfigSet(&parsed)
			build.Config = parsed.Config
		case "Party":
			build.Party = parsed.Party
		case "RichNotes":
			build.RichNotes = parsed.RichNotes
		}
	}

	return finishPassiveNodes(&build)
}
