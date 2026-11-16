package mcp

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

const (
	capSection       = 3 * 1024
	capSectionDetail = 12 * 1024
	capStats         = 2 * 1024
	capStatsDetail   = 6 * 1024
)

// fitBudget drops trailing entries of res[key] until res marshals within limit, marks the result truncated with the
// number of entries omitted, and returns that number.
func fitBudget(res map[string]any, key string, limit int, how string) int {
	size := func() int {
		b, _ := json.Marshal(res)
		return len(b)
	}
	if size() <= limit {
		return 0
	}
	res["truncated"] = true
	res["how_to_get_more"] = how
	dropped := 0
	res["omitted"] = 0
	switch list := res[key].(type) {
	case []any:
		for len(list) > 0 && size() > limit {
			list = list[:len(list)-1]
			res[key] = list
			dropped++
			res["omitted"] = dropped
		}
	case []map[string]any:
		for len(list) > 0 && size() > limit {
			list = list[:len(list)-1]
			res[key] = list
			dropped++
			res["omitted"] = dropped
		}
	}
	return dropped
}

type getBuildIn struct {
	Handle  string `json:"handle"`
	Section string `json:"section"`
	Detail  bool   `json:"detail"`
	Cursor  string `json:"cursor"`
}

var sections = []string{"summary", "tree", "items", "sockets", "skills", "config", "notes", "warnings"}

func (s *Server) getBuild(args json.RawMessage) (any, *ToolError) {
	var in getBuildIn
	if e := decode(args, schemaGetBuild, &in); e != nil {
		return nil, e
	}
	if in.Handle == "" {
		var list []map[string]any
		for _, id := range s.order {
			h := s.handles[id]
			b, e := h.parseRev(h.cur)
			if e != nil {
				return nil, e
			}
			list = append(list, map[string]any{"handle": id, "name": h.name, "revision": h.cur, "latest_revision": h.latest(),
				"class": b.Build.ClassName, "ascendancy": ascendancyName(b), "level": b.Build.Level, "exported": h.exported})
		}
		return map[string]any{"handles": nonNil(list)}, nil
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	b, e := h.parseRev(h.cur)
	if e != nil {
		return nil, e
	}
	env, e := h.envAt(h.cur)
	if e != nil {
		return nil, e
	}
	limit := capSection
	if in.Detail {
		limit = capSectionDetail
	}
	section := in.Section
	if section == "" {
		section = "summary"
	}
	res := map[string]any{"handle": h.id, "revision": h.cur, "section": section}
	switch section {
	case "summary":
		sum, e := summaryOf(h, h.cur)
		if e != nil {
			return nil, e
		}
		for k, v := range sum {
			res[k] = v
		}
		res["bandit"] = b.Build.Bandit
		res["pantheon"] = map[string]string{"major": b.Build.PantheonMajorGod, "minor": b.Build.PantheonMinorGod}
		res["item_count"] = len(equippedItems(b))
		res["group_count"] = len(activeSkills(b))
		res["versions"] = h.versions
		fitBudget(res, "warnings", limit, "get_build section=warnings")
	case "tree":
		treeSection(res, b, env, in.Detail)
		fitBudget(res, "notables", limit, "get_build section=tree detail=true")
	case "items":
		rows, spare := itemsSection(res, b, in.Detail)
		offset, e := pageCursor(in.Cursor, "i", len(rows))
		if e != nil {
			return nil, e
		}
		res["item_count"] = len(rows)
		res["items"] = rows[offset:]
		if omitted := fitBudget(res, "items", limit, ""); omitted > 0 {
			next := "i" + strconv.Itoa(len(rows)-omitted)
			res["next_cursor"] = next
			res["how_to_get_more"] = fmt.Sprintf("get_build section=items detail=%v cursor=%s (items %d-%d omitted)", in.Detail, next, len(rows)-omitted+1, len(rows))
		} else if spare != nil {
			res["unequipped"] = spare
			if dropped := fitBudget(res, "unequipped", limit, ""); dropped > 0 {
				delete(res, "how_to_get_more")
				res["unequipped_omitted"] = dropped
			}
		}
	case "sockets":
		res["sockets"] = socketsSection(b)
		fitBudget(res, "sockets", limit, "get_build section=sockets detail=true")
	case "skills":
		groups := skillsSection(b)
		offset, e := pageCursor(in.Cursor, "g", len(groups))
		if e != nil {
			return nil, e
		}
		res["group_count"] = len(groups)
		res["groups"] = groups[offset:]
		if omitted := fitBudget(res, "groups", limit, ""); omitted > 0 {
			next := "g" + strconv.Itoa(len(groups)-omitted)
			res["next_cursor"] = next
			res["how_to_get_more"] = fmt.Sprintf("get_build section=skills detail=true cursor=%s (groups %d-%d omitted)", next, len(groups)-omitted+1, len(groups))
		}
	case "config":
		res["config"] = configSection(b)
		fitBudget(res, "config", limit, "get_build section=config detail=true")
	case "notes":
		notes := b.Notes
		if len(notes) > limit-200 {
			notes = notes[:limit-200]
			res["truncated"] = true
			if !in.Detail {
				res["how_to_get_more"] = "get_build section=notes detail=true"
			}
		}
		res["notes"] = notes
	case "warnings":
		res["warnings"] = toAny(nonNil(cachedWarnings(env)))
		res["audit"] = auditOf(env)
		fitBudget(res, "warnings", limit, "get_build section=warnings detail=true")
	default:
		return nil, invalidArg(fmt.Sprintf("Unknown section %q.", section), sections)
	}
	return res, nil
}

func treeSection(res map[string]any, b *pob.PathOfBuilding, env *calculator.Environment, detail bool) {
	v := specTreeVersion(b)
	selected := map[int64]int64{}
	if spec := activeSpec(b); spec != nil {
		for _, m := range pob.SpecMasterySelections(*spec) {
			selected[m.Node] = m.Effect
		}
	}
	sockets := map[int64]int{}
	if spec := activeSpec(b); spec != nil {
		for _, sk := range spec.Sockets {
			sockets[int64(sk.NodeID)] = sk.ItemID
		}
	}
	texts := map[int]string{}
	for _, it := range b.Items.Items {
		texts[it.ID] = it.Text
	}
	var keystones, notables, masteries, asc, jewels, all []any
	small := 0
	ids := slices.Clone(b.Build.PassiveNodes)
	slices.Sort(ids)
	for _, id := range ids {
		n, ok := treeNodeByID(v, id)
		if !ok {
			continue
		}
		all = append(all, id)
		row := map[string]any{"id": id, "name": n.Name}
		switch n.Type {
		case "keystone":
			keystones = append(keystones, row)
		case "notable":
			if detail {
				row["stats"] = n.Stats
			}
			notables = append(notables, row)
		case "mastery":
			if eff, ok := selected[id]; ok {
				for _, m := range n.Node.MasteryEffects {
					if m.Effect == eff {
						row["effect"] = strings.Join(m.Stats, " / ")
					}
				}
			}
			masteries = append(masteries, row)
		case "ascendancy":
			asc = append(asc, row)
		case "jewel":
			if itemID, ok := sockets[id]; ok && itemID != 0 {
				row["item"] = calculator.PeekItem(texts[itemID]).Name
			}
			delete(row, "name")
			jewels = append(jewels, row)
		default:
			small++
		}
	}
	res["class"] = b.Build.ClassName
	res["ascendancy"] = ascendancyName(b)
	res["points"] = pointsOf(env)
	res["keystones"] = nonNil(keystones)
	res["notables"] = nonNil(notables)
	res["masteries"] = nonNil(masteries)
	res["ascendancy_nodes"] = nonNil(asc)
	res["jewel_sockets"] = nonNil(jewels)
	res["small_node_count"] = small
	res["tree_version"] = string(v)
	if detail {
		res["node_ids"] = all
	}
}

var keyModWords = []string{"maximum life", "resistance", "to all attributes", "to strength", "to dexterity", "to intelligence", "energy shield", "damage", "maximum mana", "speed", "critical"}

// keyMods ranks an item's visible mod lines life, resistances, attributes, then damage, and keeps max of them.
func keyMods(item *calculator.Item, maxLines int) []string {
	type ranked struct {
		line string
		rank int
		pos  int
	}
	var lines []ranked
	pos := 0
	for _, list := range [][]calculator.ModLine{item.ImplicitModLines, item.ExplicitModLines} {
		for _, ml := range list {
			if !item.ItemModLineVisible(ml) {
				continue
			}
			text, _, ok := calculator.FormatModLine(ml)
			if !ok || text == "" {
				text = ml.Line
			}
			r := len(keyModWords)
			lower := strings.ToLower(text)
			for i, w := range keyModWords {
				if strings.Contains(lower, w) {
					r = i
					break
				}
			}
			lines = append(lines, ranked{text, r, pos})
			pos++
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].rank < lines[j].rank })
	var out []string
	for _, l := range lines {
		if len(out) == maxLines {
			break
		}
		out = append(out, l.line)
	}
	return out
}

func socketString(item *calculator.Item) string {
	var b strings.Builder
	for i, s := range item.Sockets {
		if i > 0 {
			if s.Group == item.Sockets[i-1].Group {
				b.WriteByte('-')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(s.Color)
	}
	return b.String()
}

func itemRow(slot string, id int, text string, detail bool) map[string]any {
	item := calculator.ParseItem(text)
	row := map[string]any{"slot": slot, "item_id": id, "name": item.Name, "base": item.BaseName, "rarity": item.Rarity}
	if detail {
		row["text"] = text
	} else {
		row["key_mods"] = nonNil(keyMods(item, 4))
	}
	if s := socketString(item); s != "" {
		row["sockets"] = s
	}
	return row
}

// pageCursor reads a next_cursor value: prefix then the offset of the first row on the page.
func pageCursor(cursor, prefix string, n int) (int, *ToolError) {
	if cursor == "" {
		return 0, nil
	}
	off, err := strconv.Atoi(strings.TrimPrefix(cursor, prefix))
	if err != nil || !strings.HasPrefix(cursor, prefix) || off < 0 || off > n {
		return 0, invalidArg("cursor must be a next_cursor value from a previous page.", nil)
	}
	return off, nil
}

// itemsSection fills empty_slots and returns the equipped item rows and, with detail, the unequipped items; without
// detail it counts them in unequipped_count, as PoB keeps the item an equip replaces (ItemsTab.lua:1804-1817).
func itemsSection(res map[string]any, b *pob.PathOfBuilding, detail bool) ([]any, []any) {
	list := equippedItems(b)
	rows := make([]any, 0, len(list))
	used := map[string]bool{}
	equippedIDs := map[int]bool{}
	for _, e := range list {
		row := itemRow(e.slot, e.id, e.text, detail)
		if strings.HasPrefix(e.slot, "Flask") {
			row["active"] = e.active
		}
		rows = append(rows, row)
		used[e.slot] = true
		equippedIDs[e.id] = true
	}
	var empty []string
	for _, s := range []string{"Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt"} {
		if !used[s] {
			empty = append(empty, s)
		}
	}
	res["empty_slots"] = nonNil(empty)
	spare := []any{}
	for _, it := range b.Items.Items {
		if !equippedIDs[it.ID] {
			spare = append(spare, map[string]any{"item_id": it.ID, "name": calculator.PeekItem(it.Text).Name})
		}
	}
	if !detail {
		if len(spare) > 0 {
			res["unequipped_count"] = len(spare)
		}
		return rows, nil
	}
	return rows, spare
}

func skillsSection(b *pob.PathOfBuilding) []any {
	groups := activeSkills(b)
	loose := unsocketed(b)
	out := make([]any, 0, len(groups))
	for i, g := range groups {
		var gems []any
		for k, gem := range g.Gems {
			row := map[string]any{"index": k + 1, "name": gem.NameSpec, "level": gem.Level, "quality": gem.Quality,
				"support": gemIsSupport(gem), "enabled": gem.Enabled}
			if parts := gemParts(gem); len(parts) > 1 {
				row["part"] = parts[gemPart(gem, parts)-1]
				row["parts"] = parts
			}
			gems = append(gems, row)
		}
		row := map[string]any{"group": i + 1, "enabled": g.Enabled, "main": b.Build.MainSocketGroup == i+1, "gems": nonNil(gems),
			"in_full_dps": g.IncludeInFullDPS != nil && *g.IncludeInFullDPS}
		if g.Slot != "" {
			row["slot"] = g.Slot
		}
		if u, ok := loose[i+1]; ok {
			row["socketed"] = false
			for _, k := range []string{"note", "best_slot", "links"} {
				if v, ok := u[k]; ok {
					row[k] = v
				}
			}
		}
		if g.Label != "" {
			row["label"] = g.Label
		}
		out = append(out, row)
	}
	return nonNil(out)
}

func configSection(b *pob.PathOfBuilding) []any {
	labels := map[string]string{}
	for _, o := range configOptions() {
		labels[o.Var] = o.cleanLabel()
	}
	out := make([]any, 0, len(b.Config.Inputs))
	for _, in := range b.Config.Inputs {
		var v any
		switch {
		case in.Boolean != nil:
			v = *in.Boolean
		case in.Number != nil:
			v = *in.Number
		case in.String != nil:
			v = *in.String
		}
		out = append(out, map[string]any{"var": in.Name, "label": labels[in.Name], "value": v})
	}
	for _, block := range b.GetCustomModifierBlocks() {
		if block.Text != "" {
			out = append(out, map[string]any{"var": "customMods", "label": "Custom modifiers: " + block.BlockTitle(), "value": block.Text, "enabled": block.IsEnabled()})
		}
	}
	return nonNil(out)
}

type getStatsIn struct {
	Handle   string   `json:"handle"`
	Stats    []string `json:"stats"`
	Groups   []string `json:"groups"`
	Explain  string   `json:"explain"`
	Revision *int     `json:"revision"`
}

func (s *Server) getStats(args json.RawMessage) (any, *ToolError) {
	var in getStatsIn
	if e := decode(args, schemaGetStats, &in); e != nil {
		return nil, e
	}
	h, e := s.get(in.Handle)
	if e != nil {
		return nil, e
	}
	rev := h.cur
	if in.Revision != nil {
		rev = *in.Revision
	}
	env, e := h.envAt(rev)
	if e != nil {
		return nil, e
	}
	b, e := h.parseRev(rev)
	if e != nil {
		return nil, e
	}
	keys := slices.Clone(in.Stats)
	for _, g := range in.Groups {
		list, ok := statGroups[g]
		if !ok {
			return nil, invalidArg(fmt.Sprintf("Unknown group %q.", g), groupNames)
		}
		keys = append(keys, list...)
	}
	if len(keys) == 0 && in.Explain == "" {
		keys = summaryKeys
	}
	stats := map[string]float64{}
	var unknown []map[string]any
	var absent []string
	for _, k := range keys {
		if v, ok := statValue(env, k); ok {
			stats[k] = round(v)
		} else if _, known := statTable()[k]; known {
			stats[k] = 0
			if slices.Contains(in.Stats, k) {
				absent = append(absent, k)
			}
		} else {
			unknown = append(unknown, map[string]any{"key": k, "did_you_mean": suggest(k, statKeys(), 3)})
		}
	}
	res := map[string]any{"handle": h.id, "revision": rev, "stats": stats}
	if ms := mainSkillOf(b); ms != nil {
		res["main_skill"] = ms.Name
	}
	if len(unknown) > 0 {
		res["unknown"] = unknown
	}
	if len(absent) > 0 {
		res["not_computed"] = absent
		res["not_computed_note"] = "PoB did not calculate these for this build (shown as 0); some exist only under a condition, e.g. NetManaRegen only with a mana degen."
	}
	if in.Explain != "" {
		res["explain"] = explainStat(env, in.Explain)
	}
	limit := capStats
	if len(in.Stats) > 0 || len(in.Groups) > 1 {
		limit = capStatsDetail
	}
	if b, _ := json.Marshal(res); len(b) > limit {
		trimStats(res, limit)
	}
	return res, nil
}

func trimStats(res map[string]any, limit int) {
	stats := res["stats"].(map[string]float64)
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for len(keys) > 0 {
		if b, _ := json.Marshal(res); len(b) <= limit {
			break
		}
		delete(stats, keys[len(keys)-1])
		keys = keys[:len(keys)-1]
	}
	res["truncated"] = true
	res["how_to_get_more"] = "ask for fewer stats or groups per call"
}

// resistExplainMods are the resistance rows CalcSections.lua:1656-1671 lists, without the maximum-resistance names so
// the base total stays the resistance's own.
var resistExplainMods = map[string][]string{
	"FireResist": {"FireResist", "ElementalResist"}, "ColdResist": {"ColdResist", "ElementalResist"},
	"LightningResist": {"LightningResist", "ElementalResist"}, "ChaosResist": {"ChaosResist"},
	"FireResistTotal": {"FireResist", "ElementalResist"}, "ColdResistTotal": {"ColdResist", "ElementalResist"},
	"LightningResistTotal": {"LightningResist", "ElementalResist"}, "ChaosResistTotal": {"ChaosResist"},
}

// skillExplainMods are the rows CalcSections.lua:742-750 lists for the skill's durations, read from the main skill's
// list (cfg "skill") so the skill's own and its supports' duration modifiers are among them.
var skillExplainMods = map[string][]string{
	"Duration": {"Duration", "PrimaryDuration", "SecondaryDuration", "TertiaryDuration"}, "DurationMod": {"Duration", "PrimaryDuration", "SecondaryDuration", "TertiaryDuration"},
	"DurationSecondary": {"Duration", "PrimaryDuration", "SecondaryDuration", "TertiaryDuration"}, "DurationTertiary": {"Duration", "PrimaryDuration", "SecondaryDuration", "TertiaryDuration"},
}

// avoidExplainMods are the rows CalcSections.lua:1891-1909 lists for the avoidance and immunity stats; immunities are
// flags, listed as sources without a total.
//
//pob:block:src/Modules/CalcSections.lua:1891-1909
var avoidExplainMods = map[string][]string{
	"BlindAvoidChance":         {"AvoidBlind", "BlindImmune"},
	"ShockAvoidChance":         {"AvoidShock", "AvoidElementalAilments", "AvoidAilments", "ShockImmune", "ElementalAilmentImmune"},
	"FreezeAvoidChance":        {"AvoidFreeze", "AvoidElementalAilments", "AvoidAilments", "AvoidShockAppliesToElementalAilments", "FreezeImmune", "ElementalAilmentImmune"},
	"ChillAvoidChance":         {"AvoidChill", "AvoidElementalAilments", "AvoidAilments", "AvoidShockAppliesToElementalAilments", "ChillImmune", "ElementalAilmentImmune"},
	"IgniteAvoidChance":        {"AvoidIgnite", "AvoidElementalAilments", "AvoidAilments", "AvoidShockAppliesToElementalAilments", "IgniteImmune", "ElementalAilmentImmune"},
	"SapAvoidChance":           {"AvoidSap", "AvoidElementalAilments", "AvoidAilments", "AvoidShockAppliesToElementalAilments", "SapImmune", "ElementalAilmentImmune"},
	"BrittleAvoidChance":       {"AvoidBrittle", "AvoidElementalAilments", "AvoidAilments", "AvoidShockAppliesToElementalAilments", "BrittleImmune", "ElementalAilmentImmune"},
	"ScorchAvoidChance":        {"AvoidScorch", "AvoidElementalAilments", "AvoidAilments", "AvoidShockAppliesToElementalAilments", "ScorchImmune", "ElementalAilmentImmune"},
	"BleedAvoidChance":         {"AvoidBleed", "AvoidAilments", "BleedImmune"},
	"PoisonAvoidChance":        {"AvoidPoison", "AvoidAilments", "PoisonImmune"},
	"ImpaleAvoidChance":        {"AvoidImpale", "ImpaleImmune"},
	"CurseAvoidChance":         {"AvoidCurse", "CurseImmune"},
	"SilenceAvoidChance":       {"SilenceImmune", "AvoidCurse", "CurseImmune"},
	"InterruptStunAvoidChance": {"AvoidInterruptStun"},
	"CritExtraDamageReduction": {"ReduceCritExtraDamage"},
	"CorruptedBloodImmunity":   {"CorruptedBloodImmune"},
	"MaimImmunity":             {"MaimImmune"},
	"HinderImmunity":           {"HinderImmune"},
	"KnockbackImmunity":        {"KnockbackImmune"},
	"SelfBlindDuration":        {"SelfBlindDuration"},
}

// enemyExplainMods are the enemy-side rows CalcSections.lua:618-622 lists beside a stat's player modifiers.
var enemyExplainMods = map[string]string{"CritMultiplier": "SelfCritMultiplier", "CritChance": "SelfCritChance"}

// explainCfg picks the modifier list CalcSections.lua tabulates for a stat: offence stats read the main skill's
// list (cfg "skill", or "weapon1" for a main-hand attack, CalcSections.lua:618-641), the rest the player's mod DB.
// hand is the main-hand output table when the main skill is a weapon attack.
func explainCfg(env *calculator.Environment, key string, minion bool) (string, map[string]float64) {
	actor := env.Player
	if minion {
		actor = env.Minion.Actor
		key = strings.TrimPrefix(key, "Minion")
	}
	ms := actor.MainSkill
	if ms == nil {
		return "", nil
	}
	var hand map[string]float64
	if ms.SkillFlags[calculator.SkillFlagWeapon1Attack] {
		hand = actor.OutputTable[calculator.OutTableMainHand]
	}
	_, inHand := hand[key]
	info := statTable()[key]
	if !inHand && (info == nil || info.Group != "offence") {
		return "", hand
	}
	if hand != nil {
		return "weapon1", hand
	}
	return "skill", nil
}

// breakdownLines renders PoB's own breakdown for a stat (the calculation's breakdown table, main hand first for a
// weapon attack's per-hand stats): its text lines, table rows, and per-skill reservations.
func breakdownLines(env *calculator.Environment, key string, minion bool) []string {
	actor := env.Player
	if minion {
		actor = env.Minion.Actor
		key = strings.TrimPrefix(key, "Minion")
	}
	if actor == nil || actor.Breakdown == nil {
		return nil
	}
	data := actor.Breakdown.GetData()
	e := data[key]
	if e == nil {
		e = data["MainHand."+key]
	}
	if e == nil {
		return nil
	}
	var out []string
	if e.Label != "" {
		out = append(out, stripColor(e.Label))
	}
	for _, l := range e.Lines {
		if l = strings.TrimSpace(stripColor(l)); l != "" {
			out = append(out, l)
		}
	}
	for _, row := range e.Rows {
		var cells []string
		for _, c := range e.Columns {
			if v := stripColor(row[c.Key]); v != "" {
				cells = append(cells, c.Label+" "+v)
			}
		}
		out = append(out, strings.Join(cells, ", "))
	}
	for _, r := range e.Reservations {
		line := r.SkillName + ": " + r.Base
		for _, p := range []*string{r.Mult, r.More, r.Inc, r.Efficiency, r.EfficiencyMore} {
			if p != nil {
				line += " " + stripColor(*p)
			}
		}
		out = append(out, line+" = "+r.Total)
	}
	if len(out) > 12 {
		out = append(out[:12], fmt.Sprintf("(%d more lines)", len(out)-12))
	}
	return out
}

// explainStat summarises the stat's mod sources as PoB's breakdown tabulates them: base, inc and more totals and
// the eight largest sources, plus PoB's own breakdown lines when the calculation wrote one.
func explainStat(env *calculator.Environment, key string) map[string]any {
	v, _ := statValue(env, key)
	out := map[string]any{"stat": key, "value": round(v)}
	modNames := []string{strings.TrimPrefix(key, "Minion")}
	if info, ok := statTable()[key]; ok && len(info.ModNames) > 0 {
		modNames = info.ModNames
	}
	if names, ok := resistExplainMods[strings.TrimPrefix(key, "Minion")]; ok {
		modNames = names
	}
	if names, ok := avoidExplainMods[key]; ok {
		modNames = names
	}
	if key == mainDPS {
		out["note"] = "MainDPS is CombinedDPS of the main skill plus its minions' CombinedDPS; explain Damage or Speed for sources."
		return out
	}
	if key == "FullDPS" {
		//pob:block:src/Modules/Calcs.lua:254-255
		list := make([]map[string]string, 0, len(env.Player.SkillDPS))
		for _, sk := range env.Player.SkillDPS {
			name := sk.Name
			if sk.SkillPart != "" {
				name += " (" + sk.SkillPart + ")"
			}
			list = append(list, map[string]string{"source": name, "value": fmt.Sprintf("%g x %g", round(sk.DPS), sk.Count)})
		}
		if len(list) > 8 {
			out["more_sources"] = len(list) - 8
			list = list[:8]
		}
		out["sources"] = list
		out["note"] = "FullDPS sums TotalDPS x count over the groups included in Full DPS, each calculated in its own pass (so a skill's TotalDPS there can differ slightly from MainDPS), plus the ailment and DoT terms of that pass."
		return out
	}
	minion := strings.HasPrefix(key, "Minion") && env.Minion != nil
	cfg, hand := explainCfg(env, key, minion)
	if names, ok := skillExplainMods[strings.TrimPrefix(key, "Minion")]; ok {
		modNames, cfg = names, "skill"
	}
	if hand != nil {
		if hv, ok := hand[key]; ok {
			if _, top := statValue(env, key); !top || round(hv) != round(v) {
				out["hand_value"] = round(hv)
				out["hand"] = "main hand"
				if !top {
					out["value"] = round(hv)
				}
			}
		}
	}
	if lines := breakdownLines(env, key, minion); len(lines) > 0 {
		out["breakdown"] = lines
	}
	type src struct {
		source string
		text   string
		mag    float64
	}
	var sources []src
	var base, inc float64
	more := 1.0
	sections := []calculator.ModSectionData{
		{ModName: modNames, ModType: "BASE", Cfg: cfg}, {ModName: modNames, ModType: "INC", Cfg: cfg}, {ModName: modNames, ModType: "MORE", Cfg: cfg},
		{ModName: modNames, ModType: "FLAG", Cfg: cfg, Multi: true}}
	if enemyName, ok := enemyExplainMods[key]; ok {
		sections = append(sections, calculator.ModSectionData{ModName: []string{enemyName}, ModType: "BASE", Enemy: true})
	}
	for _, sd := range sections {
		t := sd.ModType
		rows := env.CalcsModSection(sd, minion)
		for _, r := range rows {
			var val float64
			_, _ = fmt.Sscan(strings.TrimPrefix(r.Value, "+"), &val)
			switch t {
			case "BASE":
				base += val
			case "INC":
				inc += val
			case "MORE":
				more *= 1 + val/100
			}
			name := stripColor(r.SourceName)
			if name == "" {
				name = r.Source
			}
			if r.Source != "" && name != r.Source && !strings.HasPrefix(name, r.Source) {
				name = r.Source + ": " + name
			}
			if sd.Enemy {
				name = "Enemy: " + name
			}
			unit := map[string]string{"BASE": " base", "INC": "% inc", "MORE": "% more"}[t]
			text := r.Value + unit
			if t == "FLAG" {
				text = "True"
			}
			if r.Name != "" && len(modNames) > 1 {
				text += " (" + r.Name + ")"
			}
			mag := math.Abs(val) * map[string]float64{"BASE": 1, "INC": 10, "MORE": 30}[t]
			if t == "FLAG" {
				mag = math.Inf(1)
			}
			sources = append(sources, src{name, text, mag})
		}
	}
	sort.SliceStable(sources, func(i, j int) bool { return sources[i].mag > sources[j].mag })
	var list []map[string]string
	for _, s := range sources[:min(len(sources), 8)] {
		list = append(list, map[string]string{"source": s.source, "value": s.text})
	}
	out["base"] = round(base)
	out["inc"] = round(inc)
	out["more"] = math.Round(more*1000) / 1000
	out["sources"] = nonNil(list)
	if len(sources) > 8 {
		out["more_sources"] = len(sources) - 8
	}
	if key == "CritMultiplier" {
		critMultiplierStages(out, env, hand, base)
	}
	return out
}

// critMultiplierStages shows the multiplier to two decimals with the stages CalcOffence.lua:3207-3260 computes it
// in: 1 + the base sum / 100, with no increased or more stage; PoB writes its own breakdown only when the enemy
// takes increased crit damage.
//
//pob:block:src/Modules/CalcOffence.lua:3207-3260
func critMultiplierStages(out map[string]any, env *calculator.Environment, hand map[string]float64, base float64) {
	r2 := func(x float64) float64 { return math.Round(x*100) / 100 }
	v, top := statValue(env, "CritMultiplier")
	hv, inHand := hand["CritMultiplier"]
	if inHand && !top {
		v = hv
	}
	out["value"] = r2(v)
	if _, ok := out["hand_value"]; ok {
		out["hand_value"] = r2(hv)
		v = hv
	}
	if _, ok := out["breakdown"]; ok {
		return
	}
	out["breakdown"] = []string{
		fmt.Sprintf("%g%% extra damage (the base sources, enemy rows included)", round(base)),
		fmt.Sprintf("= 1 + %g%% = %.2fx", round(base), v),
	}
}
