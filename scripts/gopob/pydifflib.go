package main

import (
	"fmt"
	"sort"
	"strings"
)

//	TIE-BREAKING    find_longest_match returns the EARLIEST longest match in a, then in b, and

type pyMatch struct{ a, b, size int }

type pyOpcode struct {
	tag            string
	i1, i2, j1, j2 int
}

type pySeqMatcher struct {
	a, b       []string
	autojunk   bool
	b2j        map[string][]int
	fullbcount map[string]int
}

func newPySeqMatcher(a, b []string, autojunk bool) *pySeqMatcher {
	m := &pySeqMatcher{a: a, b: b, autojunk: autojunk}
	m.chainB()
	return m
}

func (m *pySeqMatcher) chainB() {
	m.b2j = map[string][]int{}
	for i, elt := range m.b {
		m.b2j[elt] = append(m.b2j[elt], i)
	}
	n := len(m.b)
	if m.autojunk && n >= 200 {
		ntest := n/100 + 1
		for elt, idxs := range m.b2j {
			if len(idxs) > ntest {
				delete(m.b2j, elt)
			}
		}
	}
}

func (m *pySeqMatcher) findLongestMatch(alo, ahi, blo, bhi int) pyMatch {
	besti, bestj, bestsize := alo, blo, 0
	j2len := map[int]int{}
	for i := alo; i < ahi; i++ {
		newj2len := map[int]int{}
		for _, j := range m.b2j[m.a[i]] {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			newj2len[j] = k
			if k > bestsize {
				besti, bestj, bestsize = i-k+1, j-k+1, k
			}
		}
		j2len = newj2len
	}
	for besti > alo && bestj > blo && m.a[besti-1] == m.b[bestj-1] {
		besti, bestj, bestsize = besti-1, bestj-1, bestsize+1
	}
	for besti+bestsize < ahi && bestj+bestsize < bhi &&
		m.a[besti+bestsize] == m.b[bestj+bestsize] {
		bestsize++
	}
	return pyMatch{besti, bestj, bestsize}
}

func (m *pySeqMatcher) getMatchingBlocks() []pyMatch {
	la, lb := len(m.a), len(m.b)
	queue := [][4]int{{0, la, 0, lb}}
	var blocks []pyMatch
	for len(queue) > 0 {
		q := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		alo, ahi, blo, bhi := q[0], q[1], q[2], q[3]
		x := m.findLongestMatch(alo, ahi, blo, bhi)
		if x.size > 0 {
			blocks = append(blocks, x)
			if alo < x.a && blo < x.b {
				queue = append(queue, [4]int{alo, x.a, blo, x.b})
			}
			if x.a+x.size < ahi && x.b+x.size < bhi {
				queue = append(queue, [4]int{x.a + x.size, ahi, x.b + x.size, bhi})
			}
		}
	}
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].a != blocks[j].a {
			return blocks[i].a < blocks[j].a
		}
		if blocks[i].b != blocks[j].b {
			return blocks[i].b < blocks[j].b
		}
		return blocks[i].size < blocks[j].size
	})
	var out []pyMatch
	i1, j1, k1 := 0, 0, 0
	for _, blk := range blocks {
		if i1+k1 == blk.a && j1+k1 == blk.b {
			k1 += blk.size
			continue
		}
		if k1 > 0 {
			out = append(out, pyMatch{i1, j1, k1})
		}
		i1, j1, k1 = blk.a, blk.b, blk.size
	}
	if k1 > 0 {
		out = append(out, pyMatch{i1, j1, k1})
	}
	return append(out, pyMatch{la, lb, 0})
}

func (m *pySeqMatcher) getOpcodes() []pyOpcode {
	i, j := 0, 0
	var out []pyOpcode
	for _, blk := range m.getMatchingBlocks() {
		tag := ""
		switch {
		case i < blk.a && j < blk.b:
			tag = "replace"
		case i < blk.a:
			tag = "delete"
		case j < blk.b:
			tag = "insert"
		}
		if tag != "" {
			out = append(out, pyOpcode{tag, i, blk.a, j, blk.b})
		}
		i, j = blk.a+blk.size, blk.b+blk.size
		if blk.size > 0 {
			out = append(out, pyOpcode{"equal", blk.a, i, blk.b, j})
		}
	}
	return out
}

func (m *pySeqMatcher) getGroupedOpcodes(n int) [][]pyOpcode {
	codes := m.getOpcodes()
	if len(codes) == 0 {
		codes = []pyOpcode{{"equal", 0, 1, 0, 1}}
	}
	if codes[0].tag == "equal" {
		c := codes[0]
		codes[0] = pyOpcode{c.tag, pyMax(c.i1, c.i2-n), c.i2, pyMax(c.j1, c.j2-n), c.j2}
	}
	if last := len(codes) - 1; codes[last].tag == "equal" {
		c := codes[last]
		codes[last] = pyOpcode{c.tag, c.i1, pyMin(c.i2, c.i1+n), c.j1, pyMin(c.j2, c.j1+n)}
	}
	nn := n + n
	var groups [][]pyOpcode
	var group []pyOpcode
	for _, c := range codes {
		i1, j1 := c.i1, c.j1
		if c.tag == "equal" && c.i2-c.i1 > nn {
			group = append(group, pyOpcode{c.tag, i1, pyMin(c.i2, i1+n), j1, pyMin(c.j2, j1+n)})
			groups = append(groups, group)
			group = nil
			i1, j1 = pyMax(i1, c.i2-n), pyMax(j1, c.j2-n)
		}
		group = append(group, pyOpcode{c.tag, i1, c.i2, j1, c.j2})
	}
	if len(group) > 0 && !(len(group) == 1 && group[0].tag == "equal") {
		groups = append(groups, group)
	}
	return groups
}

func (m *pySeqMatcher) ratio() float64 {
	matches := 0
	for _, blk := range m.getMatchingBlocks() {
		matches += blk.size
	}
	return pyCalcRatio(matches, len(m.a)+len(m.b))
}

func (m *pySeqMatcher) quickRatio() float64 {
	if m.fullbcount == nil {
		m.fullbcount = map[string]int{}
		for _, elt := range m.b {
			m.fullbcount[elt]++
		}
	}
	avail := map[string]int{}
	matches := 0
	for _, elt := range m.a {
		numb, seen := avail[elt]
		if !seen {
			numb = m.fullbcount[elt]
		}
		avail[elt] = numb - 1
		if numb > 0 {
			matches++
		}
	}
	return pyCalcRatio(matches, len(m.a)+len(m.b))
}

func pyCalcRatio(matches, length int) float64 {
	if length == 0 {
		return 1.0
	}
	return 2.0 * float64(matches) / float64(length)
}

func pyMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pyMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func pyFormatRangeUnified(start, stop int) string {
	beginning := start + 1
	length := stop - start
	if length == 1 {
		return fmt.Sprintf("%d", beginning)
	}
	if length == 0 {
		beginning--
	}
	return fmt.Sprintf("%d,%d", beginning, length)
}

func pyUnifiedDiff(a, b []string, fromfile, tofile, fromdate, todate string,
	n int, lineterm string) []string {
	var out []string
	started := false
	for _, group := range newPySeqMatcher(a, b, true).getGroupedOpcodes(n) {
		if !started {
			started = true
			fd, td := "", ""
			if fromdate != "" {
				fd = "\t" + fromdate
			}
			if todate != "" {
				td = "\t" + todate
			}
			out = append(out, "--- "+fromfile+fd+lineterm, "+++ "+tofile+td+lineterm)
		}
		first, last := group[0], group[len(group)-1]
		out = append(out, fmt.Sprintf("@@ -%s +%s @@%s",
			pyFormatRangeUnified(first.i1, last.i2),
			pyFormatRangeUnified(first.j1, last.j2), lineterm))
		for _, c := range group {
			if c.tag == "equal" {
				for _, line := range a[c.i1:c.i2] {
					out = append(out, " "+line)
				}
				continue
			}
			if c.tag == "replace" || c.tag == "delete" {
				for _, line := range a[c.i1:c.i2] {
					out = append(out, "-"+line)
				}
			}
			if c.tag == "replace" || c.tag == "insert" {
				for _, line := range b[c.j1:c.j2] {
					out = append(out, "+"+line)
				}
			}
		}
	}
	return out
}

func pyOpcodeString(c pyOpcode) string {
	return fmt.Sprintf("(%s, %d, %d, %d, %d)", pyRepr(c.tag), c.i1, c.i2, c.j1, c.j2)
}

var _ = strings.Join
