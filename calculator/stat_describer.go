package calculator

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/raw"
)

var (
	statDescIndexedRe = regexp.MustCompile(`\{(\d)\}`)
	statDescSignedRe  = regexp.MustCompile(`\{(\d):(\+?)d\}`)
)

type statDescValue struct {
	v   float64
	fmt string
}

//pob:src/Modules/StatDescriber.lua:13-29
func statDescriptionScopeList(scopeName string) []*raw.StatDescriptionScope {
	var list []*raw.StatDescriptionScope
	for name := scopeName; name != ""; {
		scope := raw.StatDescriptionScopeByName(name)
		if scope == nil {
			break
		}
		list = append(list, scope)
		name = scope.Parent
	}
	return list
}

//pob:src/Modules/StatDescriber.lua:31-51
func statDescMatchLimit(variants []raw.StatDescriptionVariant, val []statDescValue, quality bool) *raw.StatDescriptionVariant {
	for i := range variants {
		desc := &variants[i]
		if !quality && desc.GemQuality {
			continue
		}
		match := true
		for j, limit := range desc.Limit {
			if limit.Not {
				if limit.Hi != nil && val[j].v == *limit.Hi {
					match = false
					break
				}
			} else if (limit.Hi != nil && val[j].v > *limit.Hi) || (limit.Lo != nil && val[j].v < *limit.Lo) {
				match = false
				break
			}
		}
		if match {
			return desc
		}
	}
	return nil
}

//pob:src/Modules/StatDescriber.lua:54-205
//nolint:gocyclo // Mirrors the Lua elseif chain.
func statDescApplySpecial(val []statDescValue, spec raw.StatDescriptionSpec) {
	if spec.V < 1 || spec.V > len(val) {
		return
	}
	v := &val[spec.V-1]
	switch spec.K {
	case "negate":
		v.v = -v.v
	case "invert_chance":
		v.v = 100 - v.v
	case "negate_and_double":
		v.v = -2 * v.v
	case "divide_by_two_0dp":
		v.v /= 2
	case "divide_by_ten_0dp":
		v.v /= 10
	case "divide_by_fifteen_0dp":
		v.v /= 15
	case "divide_by_four":
		v.v, v.fmt = v.v/4, "g"
	case "divide_by_five":
		v.v, v.fmt = v.v/5, "g"
	case "divide_by_six":
		v.v, v.fmt = v.v/6, "g"
	case "divide_by_ten_1dp_if_required", "divide_by_ten_1dp":
		v.v, v.fmt = luaRound(v.v/10, 1), "g"
	case "divide_by_twelve":
		v.v, v.fmt = v.v/12, "g"
	case "divide_by_twenty":
		v.v, v.fmt = v.v/20, "g"
	case "divide_by_one_hundred":
		v.v, v.fmt = v.v/100, "g"
	case "divide_by_one_hundred_1dp":
		v.v, v.fmt = luaRound(v.v/100, 1), "g"
	case "divide_by_one_hundred_2dp_if_required", "divide_by_one_hundred_2dp":
		v.v, v.fmt = luaRound(v.v/100, 2), "g"
	case "divide_by_one_hundred_and_negate":
		v.v, v.fmt = -luaRound(v.v/100, 1), "g"
	case "divide_by_twenty_then_double_0dp":
		v.v = luaRound(v.v/20, 0) * 2
	case "divide_by_one_thousand":
		v.v, v.fmt = v.v/1000, "g"
	case "per_minute_to_per_second":
		v.v, v.fmt = luaRound(v.v/60, 1), "g"
	case "permyriad_per_minute_to_%_per_second":
		v.v, v.fmt = luaRound(v.v/60/100, 1), "g"
	case "per_minute_to_per_second_0dp":
		v.v /= 60
	case "per_minute_to_per_second_1dp":
		v.v, v.fmt = luaRound(v.v/60, 1), "g"
	case "per_minute_to_per_second_2dp", "per_minute_to_per_second_2dp_if_required":
		v.v, v.fmt = luaRound(v.v/60, 2), "g"
	case "milliseconds_to_seconds":
		v.v, v.fmt = v.v/1000, "g"
	case "milliseconds_to_seconds_0dp":
		v.v /= 1000
	case "milliseconds_to_seconds_1dp":
		v.v, v.fmt = luaRound(v.v/1000, 1), "g"
	case "milliseconds_to_seconds_2dp", "milliseconds_to_seconds_2dp_if_required":
		v.v, v.fmt = luaRound(v.v/1000, 2), "g"
	case "deciseconds_to_seconds":
		v.v, v.fmt = v.v/10, ".2f"
	case "locations_to_metres":
		v.v, v.fmt = v.v/10, "g"
	case "30%_of_value":
		v.v *= 0.3
	case "60%_of_value":
		v.v *= 0.6
	case "multiplicative_damage_modifier":
		v.v = 100 + v.v
	case "multiplicative_permyriad_damage_modifier":
		v.v, v.fmt = 100+luaRound(v.v/100, 1), "g"
	case "multiply_by_four":
		v.v *= 4
	case "times_one_point_five":
		v.v *= 1.5
	case "times_twenty":
		v.v *= 20
	case "double":
		v.v *= 2
	case "plus_two_hundred":
		v.v += 200
	}
}

func luaFormatNumber(v float64, f string, sign bool) string {
	var s string
	switch f {
	case "g":
		s = strconv.FormatFloat(v, 'g', 6, 64)
	case ".2f":
		s = strconv.FormatFloat(v, 'f', 2, 64)
	default:
		s = strconv.FormatFloat(math.Trunc(v), 'f', 0, 64)
		if s == "-0" {
			s = "0"
		}
	}
	if sign && !strings.HasPrefix(s, "-") {
		s = "+" + s
	}
	return s
}

type DescribedStats struct {
	Lines   []string
	LineMap map[string]string
}

//pob:src/Modules/StatDescriber.lua:207-299
func DescribeStats(stats map[string]float64, scopeName string, quality bool) DescribedStats {
	scopeList := statDescriptionScopeList(scopeName)
	type described struct {
		depth, order int
		description  raw.StatDescriptor
	}
	describe := map[string]described{}
	names := make([]string, 0, len(stats))
	for s := range stats {
		names = append(names, s)
	}
	sort.Strings(names)
	for _, s := range names {
		if stats[s] == 0 {
			continue
		}
		for depth, scope := range scopeList {
			index, ok := scope.Stats[s]
			if !ok {
				continue
			}
			descriptor := scope.Descriptors[strconv.Itoa(index)]
			if len(descriptor.Variants) > 0 {
				describe[descriptor.Stats[0]] = described{depth: depth + 1, order: index, description: descriptor}
			}
			break
		}
	}

	ordered := make([]described, 0, len(describe))
	for _, d := range describe {
		ordered = append(ordered, d)
	}
	slices.SortFunc(ordered, func(a, b described) int {
		if a.depth != b.depth {
			return b.depth - a.depth
		}
		return a.order - b.order
	})

	out := DescribedStats{LineMap: map[string]string{}}
	for _, descriptor := range ordered {
		val := make([]statDescValue, len(descriptor.description.Stats))
		var stat string
		for i, s := range descriptor.description.Stats {
			if v, ok := stats[s]; ok {
				val[i].v = v
				stat = s
			}
			val[i].fmt = "d"
		}
		desc := statDescMatchLimit(descriptor.description.Variants, val, quality)
		if desc == nil {
			continue
		}
		for _, spec := range desc.Specs {
			statDescApplySpecial(val, spec)
		}
		at := func(n string) statDescValue {
			i, _ := strconv.Atoi(n)
			if i < len(val) {
				return val[i]
			}
			return statDescValue{fmt: "d"}
		}
		text := statDescIndexedRe.ReplaceAllStringFunc(desc.Text, func(m string) string {
			v := at(statDescIndexedRe.FindStringSubmatch(m)[1])
			return luaFormatNumber(v.v, v.fmt, false)
		})
		text = strings.ReplaceAll(text, "{}", luaFormatNumber(val[0].v, val[0].fmt, false))
		text = strings.ReplaceAll(text, "{:+d}", luaFormatNumber(val[0].v, val[0].fmt, false))
		text = strings.ReplaceAll(text, "{:d}", luaFormatNumber(val[0].v, val[0].fmt, false))
		text = statDescSignedRe.ReplaceAllStringFunc(text, func(m string) string {
			sub := statDescSignedRe.FindStringSubmatch(m)
			v := at(sub[1])
			return luaFormatNumber(v.v, v.fmt, sub[2] == "+")
		})
		text = strings.ReplaceAll(text, "%%", "%")
		for _, line := range strings.Split(text, `\n`) {
			if line == "" {
				continue
			}
			out.Lines = append(out.Lines, line)
			out.LineMap[line] = stat
		}
	}
	return out
}
