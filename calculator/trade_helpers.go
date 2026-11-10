package calculator

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/raw"
)

// tradeStatForm is a stat description form with what getStatDescData adds to it.
type tradeStatForm struct {
	pat          *regexp.Regexp
	text         string
	limit        [][2]any
	flags        []raw.ItemStatDescriptionFlag
	groupIndexes []int
}

type tradeStatEntry struct {
	stats []string
	forms []tradeStatForm
}

var (
	tradeGroupRe         = regexp.MustCompile(`(?s)\{.*?\}`)
	tradeGroupIndexRe    = regexp.MustCompile(`(?s)^\{(\d+):?.*\}$`)
	tradeFromToRe        = regexp.MustCompile(`(?s)\{.*?\} to \{.*?\}`)
	tradeMagicRe         = regexp.MustCompile(`[().%+\-*?\[\]^$]`)
	tradeOptionIDRe      = regexp.MustCompile(`(?s)^(.*?)\|(.+)$`)
	tradeClusterEnchant  = regexp.MustCompile(`(?s).added small passive skills grant: `)
	tradeTimelessPattern = []*regexp.Regexp{
		regexp.MustCompile(`(?s)bathed in the blood of ([0-9]+) sacrificed in the name of (.+)`),
		regexp.MustCompile(`(?s)carved to glorify ([0-9]+) new faithful converted by high templar (.+)`),
		regexp.MustCompile(`(?s)commanded leadership over ([0-9]+) warriors under (.+)`),
		regexp.MustCompile(`(?s)commissioned ([0-9]+) coins to commemorate (.+)`),
		regexp.MustCompile(`(?s)denoted service of ([0-9]+) dekhara in the akhara of (.+)`),
		regexp.MustCompile(`(?s)remembrancing ([0-9]+) songworthy deeds by the line of (.+)`),
		regexp.MustCompile(`(?s)subjugating ([0-9]+) souls in the thrall of (.+)`),
		regexp.MustCompile(`(?s)binding ([0-9]+) souls to phylacteries to sustain (.+)`),
	}
)

// tradeNumberPattern is numberPattern as the gsub replacements insert it.
const tradeNumberPattern = `%d+%.?%d*`

// asciiLower is Lua's string.lower: ASCII letters only.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

func floatp(v float64) *float64 { return &v }

// getStatDescData is the stat descriptions with each form's group indexes and pattern; nil when go-pob-data does not
// publish them.
//
//pob:src/Classes/TradeHelpers.lua:12-65
var getStatDescData = sync.OnceValue(func() []tradeStatEntry {
	descs, err := raw.ItemStatDescriptions()
	if err != nil {
		return nil
	}
	statDescData := make([]tradeStatEntry, len(descs))
	for i, d := range descs {
		entry := tradeStatEntry{stats: d.Stats, forms: make([]tradeStatForm, len(d.Forms))}
		for j, f := range d.Forms {
			var groupIndexes []int
			leftToRightIdx := 1
			for _, valueGroup := range tradeGroupRe.FindAllString(f.Text, -1) {
				if m := tradeGroupIndexRe.FindStringSubmatch(valueGroup); m != nil {
					// canonical stat is 1 indexed while group ids are zero indexed
					n, _ := strconv.Atoi(m[1])
					groupIndexes = append(groupIndexes, n+1)
				} else {
					groupIndexes = append(groupIndexes, leftToRightIdx)
				}
				leftToRightIdx++
			}
			// pob doesn't parse this as a part of other mods
			text := strings.ReplaceAll(f.Text, "\nPassage", "")
			pat := asciiLower(text)
			pat = strings.ReplaceAll(pat, "-{", "{")
			pat = strings.ReplaceAll(pat, "+{", "{")
			pat = tradeMagicRe.ReplaceAllString(pat, "%$0")
			pat = tradeFromToRe.ReplaceAllLiteralString(pat, "("+tradeNumberPattern+" to "+tradeNumberPattern+")")
			pat = tradeGroupRe.ReplaceAllLiteralString(pat, "%+?(%-?"+tradeNumberPattern+")")
			re, _ := luaPattern("^" + pat + "$")
			entry.forms[j] = tradeStatForm{text: text, limit: f.Limit, flags: f.Flags, groupIndexes: groupIndexes, pat: re}
		}
		statDescData[i] = entry
	}
	return statDescData
})

// tradeStats is Data/TradeSiteStats.lua, nil when go-pob-data does not publish it.
//
//pob:src/Classes/TradeHelpers.lua:102-104
func tradeStats() []raw.TradeStatCategory {
	stats, err := raw.TradeStats()
	if err != nil {
		return nil
	}
	return stats
}

// tradeOptionExact is an expanded option entry's stat id and its value: a number, or the option id as the site wrote
// it.
type tradeOptionExact struct {
	value   any
	tradeID string
}

type tradeOptionPattern struct {
	pattern *regexp.Regexp
	typ     string
	tradeID string
	options []raw.TradeStatOption
}

// optionTradeStatMap is getOptionTradeStatMap's exact and pattern tables; patterns keeps its keys in the order the
// trade stats first name them.
type optionTradeStatMap struct {
	exact        map[string]tradeOptionExact
	patterns     map[string]tradeOptionPattern
	patternOrder []string
}

// getOptionTradeStatMap is the trade site's option entries of the enchant, explicit and implicit categories.
//
//pob:src/Classes/TradeHelpers.lua:110-144
var getOptionTradeStatMap = sync.OnceValue(func() *optionTradeStatMap {
	m := &optionTradeStatMap{exact: map[string]tradeOptionExact{}, patterns: map[string]tradeOptionPattern{}}
	for _, cat := range tradeStats() {
		if cat.ID != "enchant" && cat.ID != "explicit" && cat.ID != "implicit" {
			continue
		}
		for _, entry := range cat.Entries {
			if ids := tradeOptionIDRe.FindStringSubmatch(entry.ID); ids != nil {
				// The 3.29 trade API expands option stats into one entry per option,
				// with the option value appended to the stat ID.
				matchKey := asciiLower(strings.ReplaceAll(entry.Text, " Passage", ""))
				var value any = ids[2]
				if n, ok := luaToNumber(ids[2]); ok {
					value = n
				}
				m.exact[cat.ID+"\x00"+matchKey] = tradeOptionExact{tradeID: ids[1], value: value}
			} else if entry.Option != nil && strings.Contains(entry.Text, "#") {
				// pob parses the passage part as a separate mod line, which
				// causes trouble
				matchKey := asciiLower(strings.ReplaceAll(strings.ReplaceAll(entry.Text, "#", "(.*)"), " Passage", ""))
				// make options lowercase
				options := make([]raw.TradeStatOption, len(entry.Option.Options))
				for i, option := range entry.Option.Options {
					options[i] = raw.TradeStatOption{ID: option.ID, Text: asciiLower(option.Text)}
				}
				if _, ok := m.patterns[matchKey]; !ok {
					m.patternOrder = append(m.patternOrder, matchKey)
				}
				// a pattern Lua cannot compile raises an error there
				re, _ := luaPattern(matchKey)
				m.patterns[matchKey] = tradeOptionPattern{pattern: re, typ: cat.ID, options: options, tradeID: entry.ID}
			}
		}
	}
	return m
})

// luaMatchFirst is string.match's first result: the first capture, or the match when the pattern has none.
func luaMatchFirst(re *regexp.Regexp, s string) (string, bool) {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	if len(m) > 1 {
		return m[1], true
	}
	return m[0], true
}

// findTradeIDOption is the trade site's option entry for a mod line, and its value when the entry has one: a number,
// or an option id as the site wrote it.
//
//pob:src/Classes/TradeHelpers.lua:183-222
func findTradeIDOption(modLine string, modType string) (string, any) {
	// match stringify() behaviour and ignore casing
	modLine = asciiLower(strings.ReplaceAll(modLine, "\n", " "))
	// exception: timeless jewels
	// these have special pseudo trade ids and are not options in poe1
	for _, pat := range tradeTimelessPattern {
		if m := pat.FindStringSubmatch(modLine); m != nil {
			value, _ := luaToNumber(m[1])
			return "explicit.pseudo_timeless_jewel_" + m[2], value
		}
	}
	if tradeStats() == nil {
		return "", nil
	}
	optionMap := getOptionTradeStatMap()

	// reformat double-line cluster enchants
	modLine = tradeClusterEnchant.ReplaceAllLiteralString(modLine, " ")
	if exactEntry, ok := optionMap.exact[modType+"\x00"+modLine]; ok {
		return exactEntry.tradeID, exactEntry.value
	}
	// PoB walks the patterns in pairs() order; the trade stats' order stands in for it
	for _, pat := range optionMap.patternOrder {
		entry := optionMap.patterns[pat]
		if entry.pattern == nil {
			continue
		}
		match, ok := luaMatchFirst(entry.pattern, modLine)
		if entry.typ == modType && ok {
			for _, option := range entry.options {
				if option.Text == match {
					return entry.tradeID, option.ID
				}
			}
		}
	}
	return "", nil
}

// impliedValue is the value a form with no variable for the canonical stat implies through its limits.
//
//pob:src/Classes/TradeHelpers.lua:229-232
func impliedValue(statForm *tradeStatForm, canonicalStat *int) *float64 {
	idx := 1
	if canonicalStat != nil {
		idx = *canonicalStat
	}
	if idx < 1 || idx > len(statForm.limit) {
		return nil
	}
	if v, ok := statForm.limit[idx-1][0].(float64); ok {
		return &v
	}
	return nil
}

//pob:src/Classes/TradeHelpers.lua:234-238
func insertUniqueHash(resultIDs []uint32, tradeHash uint32) []uint32 {
	for _, h := range resultIDs {
		if h == tradeHash {
			return resultIDs
		}
	}
	return append(resultIDs, tradeHash)
}

// tradeTimeLostJewelLines are the time-lost jewel lines that have no stat descriptor of their own.
var tradeTimeLostJewelLines = []struct{ pat, stat string }{
	{"notable passive skills in radius also grant ", "local_jewel_mod_stats_added_to_notable_passives"},
	{"small passive skills in radius also grant ", "local_jewel_mod_stats_added_to_small_passives"},
}

func flagIsOne(v any) bool {
	n, ok := v.(float64)
	return ok && n == 1
}

// findTradeHash is the trade hashes of the stat descriptions a mod line matches, more than one when the line is
// ambiguous, its value, and whether the trade site wants it negated: nil when no form set it.
//
//pob:src/Classes/TradeHelpers.lua:244-344
func findTradeHash(modLine string) ([]uint32, *float64, *bool) {
	modLine = asciiLower(modLine)
	var resultIDs []uint32
	var value *float64
	var shouldNegate *bool
	var extraStat string
	// time-lost jewels don't have proper stat descriptors and need to be handled separately
	for _, l := range tradeTimeLostJewelLines {
		if strings.HasPrefix(modLine, l.pat) {
			modLine = strings.Replace(modLine, l.pat, "", 1)
			extraStat = l.stat
			break
		}
	}
	statDescData := getStatDescData()
entries:
	for e := range statDescData {
		statDescEntry := &statDescData[e]
		statDescriptions := statDescEntry.forms
		// the stat descriptions default to using the first stat for the trade site, but this
		// flag can define it to be another one
		var canonicalStat *int
		canonicalNegated := false
		for statFormIdx, statForm := range statDescriptions {
			negate := false
			for _, flag := range statForm.flags {
				if (flag.K == "negate" || flag.K == "negate_and_double") && flagIsOne(flag.V) {
					negate = true
				}
				if flag.K == "canonical_stat" {
					if n, ok := flag.V.(float64); ok {
						canonicalStat = new(int(n))
					}
				}
				if statFormIdx == 0 || (flag.K == "canonical_line" && flag.V != nil && flag.V != false) {
					canonicalNegated = negate
				}
			}
		}
		for f := range statDescriptions {
			statForm := &statDescriptions[f]
			negate := false
			for _, flag := range statForm.flags {
				if (flag.K == "negate" || flag.K == "negate_and_double") && flagIsOne(flag.V) {
					negate = true
				}
			}
			// stat has no variables
			if modLine == asciiLower(statForm.text) {
				resultIDs = insertUniqueHash(resultIDs, hashStats(statDescEntry.stats, extraStat))
				shouldNegate = new(false)
				value = impliedValue(statForm, canonicalStat)
				continue entries
			}
			if statForm.pat == nil {
				continue
			}
			matches := statForm.pat.FindStringSubmatch(modLine)
			if matches == nil {
				continue
			}
			matchedCanonical := false
			// string.match returns the whole match when the pattern has no captures
			captures := matches[1:]
			if len(captures) == 0 {
				captures = matches
			}
			for idx, match := range captures {
				var number *float64
				if n, ok := luaToNumber(match); ok {
					number = &n
				} else if v, ok := tradeModLineValue(match); ok {
					number = &v
				}
				var group *int
				if idx < len(statForm.groupIndexes) {
					group = &statForm.groupIndexes[idx]
				}
				sameGroup := (group == nil && canonicalStat == nil) || (group != nil && canonicalStat != nil && *group == *canonicalStat)
				if (number != nil && canonicalStat == nil) || sameGroup {
					shouldNegate = new(negate != canonicalNegated)
					resultIDs = insertUniqueHash(resultIDs, hashStats(statDescEntry.stats, extraStat))
					value = number
					matchedCanonical = true
					break
				}
			}
			// the canonical stat is missing from the form text: take it
			// from the limits. this can be e.g. 100% chance omitting the
			// chance.
			if canonicalStat != nil && !matchedCanonical {
				if implied := impliedValue(statForm, canonicalStat); implied != nil {
					shouldNegate = new(negate != canonicalNegated)
					resultIDs = insertUniqueHash(resultIDs, hashStats(statDescEntry.stats, extraStat))
					value = implied
				}
			}
		}
	}
	return resultIDs, value, shouldNegate
}

// hashStats is the trade site's hash of a stat description's stats.
//
//pob:src/Modules/Common.lua:1135-1146
func hashStats(stats []string, extraStat string) uint32 {
	if extraStat != "" {
		stats = append(append([]string{}, stats...), extraStat)
	}
	statHashes := make([]byte, 0, 4*len(stats))
	for _, statName := range stats {
		statHashes = append(statHashes, intToBytes(murmurHash2([]byte(statName), 0xC58F1A7B))...)
	}
	return murmurHash2(statHashes, 0x02312233)
}

//pob:src/Modules/Common.lua:332-339
func intToBytes(v uint32) []byte {
	return []byte{byte(v & 0xFF), byte(v >> 8 & 0xFF), byte(v >> 16 & 0xFF), byte(v >> 24 & 0xFF)} //nolint:gosec // masked to a byte
}

//pob:src/Modules/Common.lua:341-372
func murmurHash2(key []byte, seed uint32) uint32 {
	mix := func(v uint32) uint32 { return v * 0x5BD1E995 }
	n := len(key)
	h := seed ^ uint32(n) //nolint:gosec // a key length
	o := 0
	for n >= 4 {
		k := uint32(key[o]) | uint32(key[o+1])<<8 | uint32(key[o+2])<<16 | uint32(key[o+3])<<24
		k = mix(k)
		k ^= k >> 24
		k = mix(k)
		h = mix(h)
		h ^= k
		o += 4
		n -= 4
	}
	if n > 0 {
		var k uint32
		for i := range n {
			k |= uint32(key[o+i]) << (8 * i)
		}
		h ^= k
		h = mix(h)
	}
	h ^= h >> 13
	h = mix(h)
	h ^= h >> 15
	return h
}
