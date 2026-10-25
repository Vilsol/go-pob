// Command changelog writes go-pob's version history to frontend/static/changelog.txt in Path of Building's
// VERSION[x][date] format, from the v* tags and the conventional-commit subjects between them, plus the version
// semantic-release would cut next. It reads .releaserc.json's conventionalcommits preset rules.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	tagPattern     = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	headerPattern  = regexp.MustCompile(`^(\w*)(?:\((.*)\))?(!?): (.*)$`)
	notePattern    = regexp.MustCompile(`(?im)^[\s|*]*(?:BREAKING CHANGE|BREAKING-CHANGE)[:\s]+(.*)$`)
	revertPattern  = regexp.MustCompile(`(?i)^(?:Revert|revert:)\s"?([\s\S]+?)"?\s*This reverts commit (\w*)\.`)
	breakingTitle  = "Breaking Changes"
	sectionAliases = map[string]string{"Features": "New", "Bug Fixes": "Fixed"}
)

// conventional-changelog-conventionalcommits src/constants.js DEFAULT_COMMIT_TYPES
var commitTypes = []struct {
	typ, section string
	hidden       bool
}{
	{"feat", "Features", false},
	{"feature", "Features", false},
	{"fix", "Bug Fixes", false},
	{"perf", "Performance Improvements", false},
	{"revert", "Reverts", false},
	{"docs", "Documentation", true},
	{"style", "Styles", true},
	{"chore", "Miscellaneous Chores", true},
	{"refactor", "Code Refactoring", true},
	{"test", "Tests", true},
	{"build", "Build System", true},
	{"ci", "Continuous Integration", true},
}

type semver [3]int

func (v semver) String() string { return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]) }

type commit struct {
	date                        string
	typ, scope, subject, header string
	revert                      bool
	notes                       []string
}

type entry struct{ scope, text string }

type version struct {
	name, date string
	sections   map[string][]entry
	notes      []entry
}

func main() {
	out := flag.String("o", "frontend/static/changelog.txt", "output file; a relative path is under the repository root")
	ref := flag.String("ref", "HEAD", "the revision whose tags and next version are written")
	flag.Parse()
	root, err := git(".", "rev-parse", "--show-toplevel")
	if err != nil {
		fail(err)
	}
	text, err := generate(root, *ref)
	if err != nil {
		fail(err)
	}
	path := *out
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil { //nolint:gosec // a static asset
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "changelog:", err)
	os.Exit(1)
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TZ=UTC")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

func generate(dir, ref string) (string, error) {
	list, err := git(dir, "tag", "--merged", ref, "--list", "v[0-9]*")
	if err != nil {
		return "", err
	}
	type tag struct {
		name string
		v    semver
	}
	var tags []tag
	for _, name := range strings.Fields(list) {
		m := tagPattern.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		var v semver
		for i := range v {
			v[i], _ = strconv.Atoi(m[i+1])
		}
		tags = append(tags, tag{name, v})
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("no v* tag is reachable from %s", ref)
	}
	sort.Slice(tags, func(i, j int) bool {
		a, b := tags[i].v, tags[j].v
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		if a[1] != b[1] {
			return a[1] < b[1]
		}
		return a[2] < b[2]
	})
	var versions []version
	for i := 1; i < len(tags); i++ {
		commits, err := readCommits(dir, tags[i-1].name+".."+tags[i].name)
		if err != nil {
			return "", err
		}
		date, err := git(dir, "log", "-1", "--format=%cd", "--date=format-local:%Y/%m/%d", tags[i].name)
		if err != nil {
			return "", err
		}
		versions = append(versions, build(tags[i].v.String(), date, commits))
	}
	last := tags[len(tags)-1]
	commits, err := readCommits(dir, last.name+".."+ref)
	if err != nil {
		return "", err
	}
	if next, ok := bump(last.v, commits); ok {
		v := build(next.String(), "", commits)
		// the release date is not known yet: the newest listed commit's date
		for _, c := range commits {
			if listed(c) {
				v.date = c.date
				break
			}
		}
		versions = append(versions, v)
	}
	var b strings.Builder
	for i := len(versions) - 1; i >= 0; i-- {
		write(&b, versions[i])
		if i > 0 {
			b.WriteString("\n\n")
		}
	}
	return b.String(), nil
}

func readCommits(dir, rangeSpec string) ([]commit, error) {
	raw, err := git(dir, "log", "--format=%cd%x1f%B%x1e", "--date=format-local:%Y/%m/%d", rangeSpec)
	if err != nil {
		return nil, err
	}
	var out []commit
	for _, rec := range strings.Split(raw, "\x1e") {
		rec = strings.TrimLeft(rec, "\n")
		if rec == "" {
			continue
		}
		date, msg, _ := strings.Cut(rec, "\x1f")
		out = append(out, parse(date, strings.TrimSpace(msg)))
	}
	return out, nil
}

// conventional-changelog-conventionalcommits src/parser.js
func parse(date, msg string) commit {
	header, body, _ := strings.Cut(msg, "\n")
	c := commit{date: date, header: header}
	bang := false
	if m := headerPattern.FindStringSubmatch(header); m != nil {
		c.typ, c.scope, c.subject, bang = m[1], m[2], m[4], m[3] == "!"
	}
	for _, m := range notePattern.FindAllStringSubmatch(body, -1) {
		c.notes = append(c.notes, strings.TrimSpace(m[1]))
	}
	if bang && len(c.notes) == 0 {
		c.notes = append(c.notes, c.subject)
	}
	c.revert = revertPattern.MatchString(msg)
	return c
}

// .releaserc.json releaseRules, then @semantic-release/commit-analyzer's default-release-rules.js
func bump(v semver, commits []commit) (semver, bool) {
	level := -1
	for _, c := range commits {
		l := -1
		switch {
		case len(c.notes) > 0: // breaking: minor, not major
			l = 1
		case c.revert:
			l = 2
		case c.typ == "feat":
			l = 1
		case c.typ == "fix" || c.typ == "perf":
			l = 2
		}
		if l >= 0 && (level < 0 || l < level) {
			level = l
		}
	}
	switch level {
	case 1:
		return semver{v[0], v[1] + 1, 0}, true
	case 2:
		return semver{v[0], v[1], v[2] + 1}, true
	}
	return v, false
}

func typeEntry(c commit) (string, bool, bool) {
	key := strings.ToLower(c.typ)
	if c.revert {
		key = "revert"
	}
	for _, t := range commitTypes {
		if t.typ == key {
			return t.section, t.hidden, true
		}
	}
	return c.typ, false, false
}

func listed(c commit) bool {
	_, hidden, found := typeEntry(c)
	return len(c.notes) > 0 || (found && !hidden)
}

// conventional-changelog-conventionalcommits src/writer.js transform
func build(name, date string, commits []commit) version {
	v := version{name: name, date: date, sections: map[string][]entry{}}
	for _, c := range commits {
		for _, n := range c.notes {
			v.notes = append(v.notes, entry{c.scope, n})
		}
		if !listed(c) {
			continue
		}
		section, _, _ := typeEntry(c)
		text := c.subject
		if text == "" {
			text = c.header
		}
		v.sections[section] = append(v.sections[section], entry{c.scope, text})
	}
	return v
}

func sectionRank(title string) int {
	for i, t := range commitTypes {
		if t.section == title {
			return i
		}
	}
	return -1
}

// writer.js commitsSort ['scope', 'subject']; unscoped entries land last, as in the published notes
func sortEntries(es []entry) {
	key := func(s string) string { return strings.ToLower(s) }
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.scope != b.scope {
			if a.scope == "" || b.scope == "" {
				return b.scope == ""
			}
			if key(a.scope) != key(b.scope) {
				return key(a.scope) < key(b.scope)
			}
			return a.scope > b.scope
		}
		if key(a.text) != key(b.text) {
			return key(a.text) < key(b.text)
		}
		return a.text > b.text
	})
}

func write(b *strings.Builder, v version) {
	fmt.Fprintf(b, "VERSION[%s][%s]\n", v.name, v.date)
	line := func(e entry) {
		if e.scope != "" {
			fmt.Fprintf(b, "* %s: %s\n", e.scope, e.text)
		} else {
			fmt.Fprintf(b, "* %s\n", e.text)
		}
	}
	if len(v.notes) > 0 {
		fmt.Fprintf(b, "\n--- %s ---\n", breakingTitle)
		for _, e := range v.notes {
			line(e)
		}
	}
	titles := make([]string, 0, len(v.sections))
	for t := range v.sections {
		titles = append(titles, t)
	}
	sort.SliceStable(titles, func(i, j int) bool {
		if ri, rj := sectionRank(titles[i]), sectionRank(titles[j]); ri != rj {
			return ri < rj
		}
		return titles[i] < titles[j]
	})
	for _, t := range titles {
		name := t
		if alias, ok := sectionAliases[t]; ok {
			name = alias
		}
		fmt.Fprintf(b, "\n--- %s ---\n", name)
		es := v.sections[t]
		sortEntries(es)
		for _, e := range es {
			line(e)
		}
	}
}
