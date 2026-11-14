package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func classNames() []string {
	out := make([]string, 0, len(pob.ClassNameByID))
	for _, n := range pob.ClassNameByID {
		out = append(out, string(n))
	}
	sort.Strings(out)
	return out
}

func resolveClass(name string) (pob.ClassName, *ToolError) {
	list := classNames()
	if c, ok, _ := resolveName(name, list, func(s string) string { return s }, nil); ok {
		return pob.ClassName(c), nil
	}
	// an ascendancy name names its class too
	for id, ascs := range pob.ClassAscendancyNames() {
		for _, a := range ascs {
			if norm(string(a)) == norm(name) {
				return pob.ClassNameByID[id], nil
			}
		}
	}
	e := errf(CodeInvalidValue, "No class named %q.", name)
	e.Allowed = toAny(list)
	return "", e
}

func resolveAscendancy(class pob.ClassName, name string) (pob.AscendancyName, *ToolError) {
	if name == "" || norm(name) == "none" {
		return "None", nil
	}
	id, _ := pob.ClassIDByName(class)
	var list []string
	for _, a := range pob.ClassAscendancyNames()[id] {
		list = append(list, string(a))
	}
	if a, ok, _ := resolveName(name, list, func(s string) string { return s }, nil); ok {
		return pob.AscendancyName(a), nil
	}
	e := errf(CodeInvalidValue, "%s has no ascendancy %q.", class, name)
	e.Allowed = toAny(append(list, "None"))
	return "", e
}

func toAny[T any](s []T) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

type newBuildIn struct {
	Class      string `json:"class"`
	Ascendancy string `json:"ascendancy"`
	Level      *int   `json:"level"`
	Name       string `json:"name"`
	FromHandle string `json:"from_handle"`
}

func (s *Server) newBuild(args json.RawMessage) (any, *ToolError) {
	var in newBuildIn
	if e := decode(args, schemaNewBuild, &in); e != nil {
		return nil, e
	}
	var b *pob.PathOfBuilding
	if in.FromHandle != "" {
		h, e := s.get(in.FromHandle)
		if e != nil {
			return nil, e
		}
		if b, e = h.parseRev(h.cur); e != nil {
			return nil, e
		}
	} else {
		if in.Class == "" {
			return nil, invalidArg("class is required.", classNames())
		}
		b = builds.EmptyBuild()
		b.ApplyNewBuildDefaults(90, 0)
		b.SetCharacterLevelAutoMode(false)
	}
	if in.Class != "" {
		class, e := resolveClass(in.Class)
		if e != nil {
			return nil, e
		}
		if class != b.Build.ClassName || in.FromHandle == "" {
			calculator.SelectClass(b, class)
		}
	}
	if in.Ascendancy != "" {
		asc, e := resolveAscendancy(b.Build.ClassName, in.Ascendancy)
		if e != nil {
			return nil, e
		}
		calculator.SelectAscendClass(b, asc)
	}
	if in.Level != nil {
		if *in.Level < 1 || *in.Level > 100 {
			e := errf(CodeInvalidValue, "level must be 1-100.")
			e.Allowed = []any{"1-100"}
			return nil, e
		}
		b.SetLevel(*in.Level)
		b.SetCharacterLevelAutoMode(false)
	}
	h, e := s.addHandle(b, in.Name, versionsOf(b, "", false))
	if e != nil {
		return nil, e
	}
	sum, e := summaryOf(h, 0)
	if e != nil {
		return nil, e
	}
	return map[string]any{
		"handle": h.id, "revision": 0, "versions": h.versions, "summary": sum,
		"next": "Add a skill with edit_skills op=add_group, allocate passives with edit_tree, gear with edit_items.",
	}, nil
}

type importIn struct {
	Source      string `json:"source"`
	ConvertTree *bool  `json:"convert_tree"`
	Name        string `json:"name"`
}

func (s *Server) importBuild(args json.RawMessage) (any, *ToolError) {
	var in importIn
	if e := decode(args, schemaImportBuild, &in); e != nil {
		return nil, e
	}
	src := strings.TrimSpace(in.Source)
	if src == "" {
		return nil, invalidArg("source is required.", schemaProps(schemaImportBuild))
	}
	xml, kind, e := fetchSource(src)
	if e != nil {
		return nil, e
	}
	switch builds.DetectGame([]byte(xml)) {
	case builds.GamePoE2, builds.GamePoE2Legacy:
		return nil, errf(CodeUnsupportedGame, "This is a Path of Exile 2 build; this server calculates PoE1 only.")
	}
	b, err := builds.ParseBuildStr(xml)
	if err != nil {
		e := errf(CodeParseError, "The build does not parse: %v", err)
		e.Hint = "Pass the full PoB export code or XML."
		return nil, e
	}
	buildTree := specTreeVersion(b)
	converted := false
	var dropped []map[string]any
	if (in.ConvertTree == nil || *in.ConvertTree) && buildTree != data.LatestTreeVersion {
		before := slices.Clone(b.Build.PassiveNodes)
		if err := calculator.ConvertSpecToVersion(b, data.LatestTreeVersion, true); err != nil {
			return nil, errf(CodeInternal, "tree conversion failed: %v", err)
		}
		converted = true
		after := map[int64]bool{}
		for _, id := range b.Build.PassiveNodes {
			after[id] = true
		}
		for _, id := range before {
			if !after[id] {
				dropped = append(dropped, map[string]any{"id": id, "name": nodeName(buildTree, id)})
			}
		}
	}
	h, e := s.addHandle(b, in.Name, versionsOf(b, buildTree, converted))
	if e != nil {
		return nil, e
	}
	sum, e := summaryOf(h, 0)
	if e != nil {
		return nil, e
	}
	conversion := map[string]any{"dropped_nodes": nonNil(dropped)}
	if len(dropped) > 30 {
		conversion["dropped_nodes"] = dropped[:30]
		conversion["more_dropped"] = len(dropped) - 30
	}
	return map[string]any{
		"handle": h.id, "revision": 0, "versions": h.versions,
		"conversion": conversion,
		"summary":    sum, "source_kind": kind,
	}, nil
}

func nodeName(v pob.TreeVersion, id int64) string {
	tv, ok := data.TreeVersions[v]
	if !ok {
		return ""
	}
	if n, ok := tv.Tree().Nodes[strconv.FormatInt(id, 10)]; ok && n.Name != nil {
		return *n.Name
	}
	return ""
}

type exportIn struct {
	Handle string `json:"handle"`
	Format string `json:"format"`
	Path   string `json:"path"`
}

// exportXML is the current revision serialised with its PlayerStat rows, as PoB saves a build.
func exportXML(h *handle) ([]byte, *ToolError) {
	b, e := h.parseRev(h.cur)
	if e != nil {
		return nil, e
	}
	env, e := h.envAt(h.cur)
	if e != nil {
		return nil, e
	}
	var player []pob.PlayerStat
	seen := map[string]bool{}
	var list []displayStat
	_ = json.Unmarshal(displayStatsJSON, &list)
	for _, d := range list {
		if seen[d.Stat] {
			continue
		}
		seen[d.Stat] = true
		if v, ok := env.Player.Output[d.Stat]; ok {
			player = append(player, pob.PlayerStat{Stat: d.Stat, Value: v})
		}
	}
	out, err := builds.SerializeBuildWithStats(b, player, nil, nil)
	if err != nil {
		return nil, errf(CodeInternal, "the build does not serialize: %v", err)
	}
	return out, nil
}

func (s *Server) exportBuild(args json.RawMessage) (any, *ToolError) {
	var in exportIn
	if e := decode(args, schemaExportBuild, &in); e != nil {
		return nil, e
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	if in.Format != "" && in.Format != "code" && in.Format != "xml" {
		e := errf(CodeInvalidValue, "format must be code or xml.")
		e.Allowed = []any{"code", "xml"}
		return nil, e
	}
	xml, e := exportXML(h)
	if e != nil {
		return nil, e
	}
	if in.Path != "" {
		if !filepath.IsAbs(in.Path) {
			return nil, invalidArg("path must be absolute.", nil)
		}
		if err := os.WriteFile(in.Path, xml, 0o600); err != nil {
			return nil, errf(CodeInvalidValue, "cannot write %s: %v", in.Path, err)
		}
	}
	file, e := s.recordExport(h, xml)
	if e != nil {
		return nil, e
	}
	h.exported = true
	sum, e := summaryOf(h, h.cur)
	if e != nil {
		return nil, e
	}
	out := map[string]any{"handle": h.id, "revision": h.cur, "bytes": len(xml), "summary": compactSummary(sum)}
	if file != "" {
		out["export_file"] = file
	}
	if in.Format == "xml" {
		out["xml"] = string(xml)
	} else {
		code, err := pob.CompressEncode(string(xml))
		if err != nil {
			return nil, errf(CodeInternal, "the build does not encode: %v", err)
		}
		out["code"] = code
	}
	return out, nil
}

func compactSummary(sum map[string]any) map[string]any {
	return map[string]any{"class": sum["class"], "ascendancy": sum["ascendancy"], "level": sum["level"], "stats": sum["stats"]}
}

// recordExport writes one PoB XML file per export into the export directory, named so a directory listing
// sorts in export order: <seq>-<handle>-r<revision>.xml.
func (s *Server) recordExport(h *handle, xml []byte) (string, *ToolError) {
	dir := s.opts.ExportDir
	if dir == "" {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", errf(CodeInternal, "cannot create the export directory: %v", err)
	}
	s.exportSeq++
	name := filepath.Join(dir, fmt.Sprintf("%04d-%s-r%d.xml", s.exportSeq, h.id, h.cur))
	for {
		if _, err := os.Stat(name); err != nil {
			break
		}
		s.exportSeq++
		name = filepath.Join(dir, fmt.Sprintf("%04d-%s-r%d.xml", s.exportSeq, h.id, h.cur))
	}
	if err := os.WriteFile(name, xml, 0o600); err != nil {
		return "", errf(CodeInternal, "cannot write the export: %v", err)
	}
	return name, nil
}

type revertIn struct {
	Handle     string `json:"handle"`
	ToRevision *int   `json:"to_revision"`
}

func (s *Server) revertBuild(args json.RawMessage) (any, *ToolError) {
	var in revertIn
	if e := decode(args, schemaRevertBuild, &in); e != nil {
		return nil, e
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	to := h.cur - 1
	if in.ToRevision != nil {
		to = *in.ToRevision
	}
	if to < 0 {
		e := errf(CodeInvalidValue, "Handle %s is at revision 0; there is nothing before it.", h.id)
		e.Allowed = []any{fmt.Sprintf("0-%d", h.latest())}
		return nil, e
	}
	before, e := h.envAt(h.cur)
	if e != nil {
		return nil, e
	}
	after, e := h.envAt(to)
	if e != nil {
		return nil, e
	}
	from := h.cur
	h.cur = to
	res := deltaResult(h, from, before, after, false, []string{fmt.Sprintf("revert to revision %d", to)}, nil)
	res["revision"] = to
	res["latest_revision"] = h.latest()
	capWriteResult(res)
	return res, nil
}
