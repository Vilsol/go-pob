package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
)

// poe2Describer is PoB-PoE2's Export/statdesc.lua: one stat description file and its includes.
type poe2Describer struct {
	descriptors map[string]*edDescriptor
}

var (
	poe2IncludeRe     = regexp.MustCompile(`^Data/StatDescriptions/(.+)\.csd$`)
	poe2EscapeRe      = regexp.MustCompile(`\\([^nb])`)
	poe2SignedRe      = regexp.MustCompile(`\{(\d?):([+-]?)d?\}`)
	poe2LineRe        = regexp.MustCompile(`([^\\]+)\\n`)
	poe2AngleRe       = regexp.MustCompile(`<[^<>]*>`)
	poe2SanitiseFirst = regexp.MustCompile(`[\x{80}-\x{10FFFF}<]`)
)

//pob2:src/Export/statdesc.lua:5-114
func poe2LoadStatFile(name string) (*poe2Describer, error) {
	e := &poe2Describer{descriptors: map[string]*edDescriptor{}}
	if _, err := e.parseStatFile(name, 1); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *poe2Describer) parseStatFile(name string, order int) (int, error) {
	file, err := loader.LoadTranslation(gameDataContext(), poe2Version, "en", name, assetCache())
	if err != nil {
		return order, fmt.Errorf("%s: %w", name, err)
	}
	for _, inc := range file.Includes {
		m := poe2IncludeRe.FindStringSubmatch(inc)
		if m == nil {
			return order, fmt.Errorf("%s: include %q is not a stat description file", name, inc)
		}
		if order, err = e.parseStatFile(m[1], order); err != nil {
			return order, err
		}
	}
	for _, stat := range file.NoDescription {
		e.descriptors[stat] = &edDescriptor{}
	}
	for _, d := range file.Descriptors {
		desc := &edDescriptor{stats: d.IDs, order: order}
		order++
		for _, line := range d.List {
			if strings.Contains(line.Qualifier, "table_only") {
				continue
			}
			text := poe2EscapeRe.ReplaceAllString(escapeGGGString(line.String), `\n${1}`)
			v := sdVariant{Text: text, Specs: sdSpecs(line.Handlers), GemQuality: strings.Contains(line.Qualifier, "gem_quality")}
			for _, c := range line.Conditions {
				v.Limit = append(v.Limit, sdLimit(c))
			}
			desc.variants = append(desc.variants, v)
		}
		for _, stat := range d.IDs {
			e.descriptors[stat] = desc
		}
	}
	return order, nil
}

//pob2:block:src/Export/statdesc.lua:205-377
//nolint:gocyclo // Mirrors the Lua elseif chain.
func poe2ApplySpec(val []*edRange, spec sdSpec) error {
	switch spec.K {
	case "reminderstring", "canonical_line", "canonical_stat":
		return nil
	}
	idx, ok := spec.V.(float64)
	if !ok || int(idx) < 1 || int(idx) > len(val) {
		return fmt.Errorf("handler %s indexes stat %v of %d", spec.K, spec.V, len(val))
	}
	v := val[int(idx)-1]
	both := func(f func(float64) float64, format string) {
		v.min, v.max = f(v.min), f(v.max)
		if format != "" {
			v.fmt = format
		}
	}
	switch spec.K {
	case "negate":
		v.max, v.min = -v.min, -v.max
	case "invert_chance":
		v.max, v.min = 100-v.min, 100-v.max
	case "negate_and_double":
		v.max, v.min = -2*v.min, -2*v.max
	case "passive_hash":
		if v.min < 0 {
			v.min += 65536
			v.max += 65536
		}
	case "divide_by_two_0dp":
		both(func(x float64) float64 { return luaRound(x/2, 0) }, "")
	case "divide_by_three":
		v.min /= 3
		v.max = v.min / 3
		v.fmt = "g"
	case "divide_by_four":
		v.min /= 4
		v.max = v.min / 4
		v.fmt = "g"
	case "divide_by_five":
		both(func(x float64) float64 { return x / 5 }, "g")
	case "divide_by_six":
		both(func(x float64) float64 { return x / 6 }, "g")
	case "divide_by_ten_0dp":
		both(func(x float64) float64 { return luaRound(x/10, 0) }, "")
	case "divide_by_ten_1dp", "divide_by_ten_1dp_if_required":
		both(func(x float64) float64 { return luaRound(x/10, 1) }, "g")
	case "divide_by_twelve":
		both(func(x float64) float64 { return x / 12 }, "g")
	case "divide_by_fifteen_0dp":
		both(func(x float64) float64 { return luaRound(x/15, 0) }, "")
	case "divide_by_twenty":
		both(func(x float64) float64 { return x / 20 }, "g")
	case "divide_by_twenty_then_double_0dp":
		both(func(x float64) float64 { return luaRound(x/20, 0) * 2 }, "")
	case "divide_by_fifty":
		both(func(x float64) float64 { return x / 50 }, "g")
	case "divide_by_one_hundred":
		both(func(x float64) float64 { return x / 100 }, "g")
	case "divide_by_one_hundred_0dp":
		both(func(x float64) float64 { return luaRound(x/100, 0) }, "g")
	case "divide_by_one_hundred_1dp":
		both(func(x float64) float64 { return luaRound(x/100, 1) }, "g")
	case "divide_by_one_hundred_2dp_if_required", "divide_by_one_hundred_2dp":
		both(func(x float64) float64 { return luaRound(x/100, 2) }, "g")
	case "divide_by_one_hundred_and_negate":
		both(func(x float64) float64 { return -x / 100 }, "g")
	case "divide_by_one_thousand":
		both(func(x float64) float64 { return x / 1000 }, "g")
	case "divide_by_ten_thousand_1dp":
		both(func(x float64) float64 { return luaRound(x/10000, 1) }, "g")
	case "per_minute_to_per_second":
		both(func(x float64) float64 { return x / 60 }, "g")
	case "per_minute_to_per_second_0dp":
		both(func(x float64) float64 { return luaRound(x/60, 0) }, "")
	case "per_minute_to_per_second_1dp":
		both(func(x float64) float64 { return luaRound(x/60, 1) }, "g")
	case "per_minute_to_per_second_2dp_if_required", "per_minute_to_per_second_2dp":
		both(func(x float64) float64 { return luaRound(x/60, 2) }, "g")
	case "milliseconds_to_seconds":
		both(func(x float64) float64 { return x / 1000 }, "g")
	case "milliseconds_to_seconds_halved":
		both(func(x float64) float64 { return x / 1000 / 2 }, "g")
	case "milliseconds_to_seconds_0dp":
		both(func(x float64) float64 { return luaRound(x/1000, 0) }, "")
	case "milliseconds_to_seconds_1dp":
		both(func(x float64) float64 { return luaRound(x/1000, 1) }, "g")
	case "milliseconds_to_seconds_2dp", "milliseconds_to_seconds_2dp_if_required":
		both(func(x float64) float64 { return luaRound(x/1000, 2) }, "g")
	case "deciseconds_to_seconds":
		both(func(x float64) float64 { return x / 10 }, ".2f")
	case "30%_of_value":
		both(func(x float64) float64 { return x * 0.3 }, "")
	case "60%_of_value":
		both(func(x float64) float64 { return x * 0.6 }, "")
	case "mod_value_to_item_class":
		return fmt.Errorf("handler %s needs ItemClasses, which no ported export reads", spec.K)
	case "one_hundred_divide_by_value":
		both(func(x float64) float64 { return luaRound(100/x, 2) }, "g")
	case "multiplicative_damage_modifier":
		both(func(x float64) float64 { return 100 + x }, "")
	case "multiplicative_permyriad_damage_modifier":
		both(func(x float64) float64 { return 100 + x/100 }, "g")
	case "times_one_point_five":
		both(func(x float64) float64 { return x * 1.5 }, "")
	case "double":
		both(func(x float64) float64 { return x * 2 }, "")
	case "multiply_by_four":
		both(func(x float64) float64 { return x * 4 }, "")
	case "multiply_by_four_and_negate":
		both(func(x float64) float64 { return -x * 4 }, "")
	case "multiply_by_ten":
		both(func(x float64) float64 { return x * 10 }, "")
	case "times_twenty":
		both(func(x float64) float64 { return x * 20 }, "")
	case "multiply_by_one_hundred":
		both(func(x float64) float64 { return x * 100 }, "")
	case "plus_two_hundred":
		both(func(x float64) float64 { return x + 200 }, "")
	}
	return nil
}

//pob2:src/Modules/Common.lua:321-345
func poe2SanitiseText(text string) string {
	if !poe2SanitiseFirst.MatchString(text) {
		return text
	}
	text = poe2AngleRe.ReplaceAllString(text, "")
	for _, r := range []struct{ from, to string }{
		{"‐", "-"}, {"‑", "-"}, {"‒", "-"}, {"–", "-"}, {"—", "-"}, {"―", "-"}, {"−", "-"},
		{"• ", ""}, {"•", ""}, {"ä", "a"}, {"ö", "o"}, {"í", "i"}, {"ó", "o"},
	} {
		text = strings.ReplaceAll(text, r.from, r.to)
	}
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == 0x96 || c == 0x97:
			b.WriteByte('-')
		case c == 0xe4:
			b.WriteByte('a')
		case c == 0xf6:
			b.WriteByte('o')
		case c >= 0x80:
			b.WriteByte('?')
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

//pob2:src/Export/statdesc.lua:157-418
func (e *poe2Describer) describeStats(stats map[string]*edRange) ([]string, []float64, error) {
	var descOrdered []*edDescriptor
	seen := map[*edDescriptor]bool{}
	for s, v := range stats {
		d := e.descriptors[s]
		if s != "Type" && d != nil && d.stats != nil && (v.min != 0 || v.max != 0) && !seen[d] {
			seen[d] = true
			descOrdered = append(descOrdered, d)
		}
	}
	sort.Slice(descOrdered, func(i, j int) bool { return descOrdered[i].order < descOrdered[j].order })

	var out []string
	var orders []float64
	for _, descriptor := range descOrdered {
		val := make([]*edRange, len(descriptor.stats))
		for i, s := range descriptor.stats {
			val[i] = stats[s]
			if val[i] == nil {
				val[i] = &edRange{}
			}
			val[i].fmt = "d"
		}
		desc := edMatchLimit(descriptor.variants, val)
		// Hack to handle ranges starting or ending at 0 where no descriptor is defined for 0
		// Attempt to adapt existing ranges
		if desc == nil {
			for _, s := range val {
				if s.min == 0 && s.max > 0 {
					s.min, s.minZ = 1, true
				} else if s.min < 0 && s.max == 0 {
					s.max, s.maxZ = -1, true
				}
			}
			desc = edMatchLimit(descriptor.variants, val)
			for _, s := range val {
				if s.minZ {
					s.min = 0
				}
				if s.maxZ {
					s.max = 0
				}
			}
		}
		if desc == nil {
			continue
		}
		for _, spec := range desc.Specs {
			if err := poe2ApplySpec(val, spec); err != nil {
				return nil, nil, fmt.Errorf("%s: %w", strings.Join(descriptor.stats, " "), err)
			}
		}
		at := func(n string) (*edRange, error) {
			i := 0
			if n != "" {
				i, _ = strconv.Atoi(n)
			}
			if i >= len(val) {
				return nil, fmt.Errorf("%s: %q reads value %d of %d", strings.Join(descriptor.stats, " "), desc.Text, i+1, len(val))
			}
			return val[i], nil
		}
		var fail error
		statDesc := edIndexedRe.ReplaceAllStringFunc(desc.Text, func(m string) string {
			v, err := at(edIndexedRe.FindStringSubmatch(m)[1])
			if err != nil {
				fail = err
				return m
			}
			return edValue(v)
		})
		statDesc = strings.ReplaceAll(statDesc, "{}", edValue(val[0]))
		statDesc = poe2SignedRe.ReplaceAllStringFunc(statDesc, func(m string) string {
			sub := poe2SignedRe.FindStringSubmatch(m)
			v, err := at(sub[1])
			if err != nil {
				fail = err
				return m
			}
			switch {
			case v.min == v.max:
				return edFormat(v.min, v.fmt, sub[2] == "+")
			case sub[2] == "+" || sub[2] == "-":
				if v.max < 0 {
					return "-(" + edFormat(-v.min, v.fmt, false) + "-" + edFormat(-v.max, v.fmt, false) + ")"
				}
				return "+(" + edFormat(v.min, v.fmt, false) + "-" + edFormat(v.max, v.fmt, false) + ")"
			default:
				return "(" + edFormat(v.min, v.fmt, false) + "-" + edFormat(v.max, v.fmt, false) + ")"
			}
		})
		if fail != nil {
			return nil, nil, fail
		}
		statDesc = strings.ReplaceAll(statDesc, "%%", "%")
		order := float64(descriptor.order)
		for _, m := range poe2LineRe.FindAllStringSubmatch(statDesc+`\n`, -1) {
			out = append(out, poe2SanitiseText(m[1]))
			orders = append(orders, order)
			order += 0.1
		}
	}
	return out, orders, nil
}
