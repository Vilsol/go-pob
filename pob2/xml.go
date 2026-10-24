package pob2

import (
	"fmt"
	"slices"
	"strings"
)

// Node is one element of xml.lua's tree: a kid with Elem == "" is a text node.
type Node struct {
	Elem   string
	Attrib map[string]string
	Kids   []*Node
	Text   string
}

func isLuaSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

func isLuaAlnum(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func trimLuaSpace(s string) string {
	i, j := 0, len(s)
	for i < j && isLuaSpace(s[i]) {
		i++
	}
	for j > i && isLuaSpace(s[j-1]) {
		j--
	}
	return s[i:j]
}

func hasNonSpace(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isLuaSpace(s[i]) {
			return true
		}
	}
	return false
}

var xmlEntities = map[string]string{"lt": "<", "gt": ">", "amp": "&", "apos": "'", "quot": `"`}

//pob2:runtime/lua/xml.lua:8-14
func decodeContent(content string) string {
	if !strings.Contains(content, "&") {
		return content
	}
	var b strings.Builder
	for {
		i := strings.IndexByte(content, '&')
		if i < 0 {
			b.WriteString(content)
			return b.String()
		}
		j := strings.IndexByte(content[i+1:], ';')
		if j < 0 {
			b.WriteString(content)
			return b.String()
		}
		b.WriteString(content[:i])
		b.WriteString(xmlEntities[content[i+1:i+1+j]])
		content = content[i+2+j:]
	}
}

//pob2:runtime/lua/xml.lua:16-22
func encodeContent(text string) string {
	if !strings.ContainsAny(text, `<>&'"`) {
		return text
	}
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '\'':
			b.WriteString("&apos;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteByte(text[i])
		}
	}
	return b.String()
}

// (%w+)=(["'])(.-)%2, scanned left to right as gmatch does.
//
//pob2:runtime/lua/xml.lua:24-30
func parseAttribs(str string) map[string]string {
	attrib := map[string]string{}
	for i := 0; i < len(str); {
		if !isLuaAlnum(str[i]) {
			i++
			continue
		}
		k := i
		for k < len(str) && isLuaAlnum(str[k]) {
			k++
		}
		if k+1 < len(str) && str[k] == '=' && (str[k+1] == '"' || str[k+1] == '\'') {
			if end := strings.IndexByte(str[k+2:], str[k+1]); end >= 0 {
				attrib[str[i:k]] = decodeContent(str[k+2 : k+2+end])
				i = k + 3 + end
				continue
			}
		}
		i++
	}
	return attrib
}

var cdataOpen = "<![CDATA["

//pob2:runtime/lua/xml.lua:34-95
func ParseXML(str string) (*Node, error) {
	for {
		i := strings.Index(str, "<!--")
		if i < 0 {
			break
		}
		j := strings.Index(str[i+4:], "-->")
		if j < 0 {
			break
		}
		str = str[:i] + str[i+4+j+3:]
	}
	top := &Node{}
	stack := []*Node{top}
	i := 0
	for {
		lt := strings.IndexByte(str[i:], '<')
		if lt < 0 {
			break
		}
		lt += i
		content := trimLuaSpace(str[i:lt])
		if strings.HasPrefix(str[lt:], cdataOpen) {
			if end := strings.Index(str[lt+len(cdataOpen):], "]]>"); end >= 0 {
				cdata := str[lt+len(cdataOpen) : lt+len(cdataOpen)+end]
				if hasNonSpace(content) {
					top.Kids = append(top.Kids, &Node{Text: decodeContent(content)})
				}
				if hasNonSpace(cdata) {
					top.Kids = append(top.Kids, &Node{Text: cdata})
				}
				i = lt + len(cdataOpen) + end + 3
				continue
			}
		}
		p := lt + 1
		cs := byte(0)
		if p < len(str) && (str[p] == '?' || str[p] == '/') {
			cs = str[p]
			p++
		}
		e := p
		for e < len(str) && (isLuaAlnum(str[e]) || str[e] == ':') {
			e++
		}
		if e == p {
			break
		}
		gt := strings.IndexByte(str[e:], '>')
		if gt < 0 {
			break
		}
		gt += e
		elem, attrib, ce := str[p:e], str[e:gt], byte(0)
		if gt > e && (str[gt-1] == '?' || str[gt-1] == '/') {
			attrib, ce = str[e:gt-1], str[gt-1]
		}
		if hasNonSpace(content) {
			top.Kids = append(top.Kids, &Node{Text: decodeContent(content)})
		}
		switch {
		case cs == '?' || ce == '?':
		case ce == '/':
			top.Kids = append(top.Kids, &Node{Elem: elem, Attrib: parseAttribs(attrib)})
		case cs == 0:
			top = &Node{Elem: elem, Attrib: parseAttribs(attrib)}
			stack = append(stack, top)
		default:
			if len(stack) == 1 {
				return nil, fmt.Errorf("nothing to close with '%s'", elem)
			}
			if top.Elem != elem {
				return nil, fmt.Errorf("trying to close <%s> with '%s'", top.Elem, elem)
			}
			closed := top
			stack = stack[:len(stack)-1]
			top = stack[len(stack)-1]
			top.Kids = append(top.Kids, closed)
		}
		i = gt + 1
	}
	if len(stack) > 1 {
		return nil, fmt.Errorf("unclosed element '%s'", stack[len(stack)-1].Elem)
	}
	if rest := trimLuaSpace(str[i:]); hasNonSpace(rest) {
		stack[0].Kids = append(stack[0].Kids, &Node{Text: decodeContent(rest)})
	}
	return stack[0], nil
}

//pob2:runtime/lua/xml.lua:115-169
func composeNode(b *strings.Builder, node *Node, lvl int) error {
	if node.Elem == "" {
		return fmt.Errorf("invalid xml tree (missing element name)")
	}
	b.WriteString(strings.Repeat("\t", lvl))
	b.WriteByte('<')
	b.WriteString(node.Elem)
	keys := make([]string, 0, len(node.Attrib))
	for k := range node.Attrib {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(encodeContent(node.Attrib[k]))
		b.WriteByte('"')
	}
	if len(node.Kids) == 0 {
		b.WriteString("/>\n")
		return nil
	}
	b.WriteString(">\n")
	for _, n := range node.Kids {
		if n.Elem != "" {
			if err := composeNode(b, n, lvl+1); err != nil {
				return err
			}
			continue
		}
		b.WriteString(strings.Repeat("\t", lvl+1))
		b.WriteString(encodeContent(n.Text))
		b.WriteByte('\n')
	}
	b.WriteString(strings.Repeat("\t", lvl))
	b.WriteString("</")
	b.WriteString(node.Elem)
	b.WriteString(">\n")
	return nil
}

//pob2:runtime/lua/xml.lua:171-182
func ComposeXML(root *Node) (string, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	if err := composeNode(&b, root, 0); err != nil {
		return "", err
	}
	return b.String(), nil
}
