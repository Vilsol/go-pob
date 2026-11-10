package calculator2

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
)

// tradeStatForm is a stat description form with what getStatDescData adds to it.
type tradeStatForm struct {
	pat          *regexp.Regexp
	text         string
	limit        [][2]any
	flags        []poe2.StatDescriptionFlag
	groupIndexes []int
}

type tradeStatEntry struct {
	stats []string
	forms []tradeStatForm
}

var (
	tradeGroupRe      = regexp.MustCompile(`(?s)\{.*?\}`)
	tradeGroupIndexRe = regexp.MustCompile(`(?s)^\{(\d+):?.*\}$`)
	tradeFromToRe     = regexp.MustCompile(`(?s)\{.*?\} to \{.*?\}`)
	tradeMagicRe      = regexp.MustCompile(`[().%+\-*?\[\]^$]`)
	tradeNumberRe     = regexp.MustCompile(`-?[0-9]+\.?[0-9]*`)
	tradeFromToValRe  = regexp.MustCompile(`(-?[0-9]+\.?[0-9]*) to (-?[0-9]+\.?[0-9]*)`)
	tradeOptionIDRe   = regexp.MustCompile(`^(.*)\|(.*)$`)
)

// tradeNumberPattern is numberPattern as the gsub replacements insert it.
const tradeNumberPattern = `%d+%.?%d*`

// getStatDescData is the stat descriptions with each form's group indexes and pattern; nil when go-pob-data does not
// publish them.
//
//pob2:src/Classes/TradeHelpers.lua:8-65
var getStatDescData = sync.OnceValue(func() []tradeStatEntry {
	descs, err := poe2.StatDescriptions()
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

// modLineValue is the first number of a mod line, or the midpoint of a "# to #" one.
//
//pob2:src/Classes/TradeHelpers.lua:91-99
func modLineValue(line string, onlyFromTo bool) *float64 {
	if m := tradeFromToValRe.FindStringSubmatch(line); m != nil {
		low, _ := strconv.ParseFloat(m[1], 64)
		high, _ := strconv.ParseFloat(m[2], 64)
		return floatp((low + high) / 2)
	}
	if onlyFromTo {
		return nil
	}
	if v, ok := luaToNumber(tradeNumberRe.FindString(line)); ok {
		return &v
	}
	return nil
}

// tradeStats is Data/TradeSiteStats.lua, nil when go-pob-data does not publish it.
//
//pob2:src/Classes/TradeHelpers.lua:102-104
func tradeStats() []poe2.TradeStatCategory {
	stats, err := poe2.TradeStats()
	if err != nil {
		return nil
	}
	return stats
}

type optionTradeStat struct {
	pattern    *regexp.Regexp
	typ        string
	text       string
	tradeID    string
	hasPattern bool
}

// getOptionTradeStatMap is the trade site's option entries of the enchant, explicit and implicit categories.
//
//pob2:src/Classes/TradeHelpers.lua:110-132
var getOptionTradeStatMap = sync.OnceValue(func() map[string][]optionTradeStat {
	optionTradeStatMap := map[string][]optionTradeStat{}
	for _, cat := range tradeStats() {
		if cat.ID != "enchant" && cat.ID != "explicit" && cat.ID != "implicit" {
			continue
		}
		optionTradeStatMap[cat.ID] = []optionTradeStat{}
		for _, entry := range cat.Entries {
			if !tradeOptionIDRe.MatchString(entry.ID) {
				continue
			}
			newEntry := optionTradeStat{typ: cat.ID, text: entry.Text, tradeID: entry.ID}
			if strings.Contains(entry.Text, "#") {
				// work around issue where pob splits timeless jewel
				// mods into separate mod lines
				pattern, _, _ := strings.Cut(entry.Text, "\n")
				pattern = strings.ReplaceAll(pattern, "+", "%+")
				pattern = strings.ReplaceAll(pattern, "#", "(%d+)")
				newEntry.pattern, _ = luaPattern(asciiLower(pattern))
				newEntry.hasPattern = true
			}
			optionTradeStatMap[cat.ID] = append(optionTradeStatMap[cat.ID], newEntry)
		}
	}
	return optionTradeStatMap
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

// findTradeIDOption is the trade site's option entry for a mod line, and its value when the entry has one.
//
//pob2:src/Classes/TradeHelpers.lua:171-188
func findTradeIDOption(modLine string, modType string) (string, *float64) {
	// match stringify() behaviour
	modLine = strings.ReplaceAll(modLine, "\n", " ")
	if tradeStats() == nil {
		return "", nil
	}
	lower := asciiLower(modLine)
	for _, v := range getOptionTradeStatMap()[modType] {
		if v.hasPattern {
			// a pattern Lua cannot compile raises an error there
			if v.pattern == nil {
				continue
			}
			if match, ok := luaMatchFirst(v.pattern, lower); ok {
				if n, ok := luaToNumber(match); ok {
					return v.tradeID, &n
				}
				return v.tradeID, nil
			}
		} else if asciiLower(v.text) == lower {
			return v.tradeID, nil
		}
	}
	return "", nil
}

// impliedValue is the value a form with no variable for the canonical stat implies through its limits.
//
//pob2:src/Classes/TradeHelpers.lua:195-198
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

//pob2:src/Classes/TradeHelpers.lua:200-204
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
//pob2:src/Classes/TradeHelpers.lua:210-310
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
				} else {
					number = modLineValue(match, false)
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
//pob2:src/Modules/Common.lua:1107-1118
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

//pob2:src/Modules/Common.lua:350-357
func intToBytes(v uint32) []byte {
	return []byte{byte(v & 0xFF), byte(v >> 8 & 0xFF), byte(v >> 16 & 0xFF), byte(v >> 24 & 0xFF)} //nolint:gosec // masked to a byte
}

//pob2:src/Modules/Common.lua:359-390
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
