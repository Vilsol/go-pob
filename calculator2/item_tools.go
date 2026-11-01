package calculator2

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	applyRangeRangeRe     = regexp.MustCompile(`([+-]?)\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)
	applyRangeAntonymRe   = regexp.MustCompile(`-(\d+\.?\d*%) ([A-Za-z]+)`)
	applyRangeNumberRe    = regexp.MustCompile(`-?\d+\.?\d*`)
	applyRangePlusRangeRe = regexp.MustCompile(`(\+?)\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)
	applyRangeIntAntonyRe = regexp.MustCompile(`-(\d+%) ([A-Za-z]+)`)
	applyRangeDecimalRe   = regexp.MustCompile(`\d+\.\d*`)
	applyRangeScalableRe  = regexp.MustCompile(`\d+\.?\d*%? `)
	valueScalarPrecRe     = regexp.MustCompile(`\d+\.?\d*`)
	valueScalarIntRe      = regexp.MustCompile(`(\d+)([^.])`)

	zeroLineStartRe = regexp.MustCompile(`^\+?0%? `)
	zeroLineMidRe   = regexp.MustCompile(` \+?0%? `)
	zeroLineToRe    = regexp.MustCompile(`0 to [1-9]`)
	zeroLinePctToRe = regexp.MustCompile(`0% to \d+%`)

	gggTagRe     = regexp.MustCompile(`<[^>]+>\{([^}]+)\}`)
	gggLinkRe    = regexp.MustCompile(`\[([^|\]]+)\]`)
	gggAltLinkRe = regexp.MustCompile(`\[[^|]+\|([^|]+)\]`)
)

// luaNum is Lua's tostring of a number (%.14g).
func luaNum(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'g', 14, 64)
}

func luaToNumber(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v, err == nil
}

// gsubFunc is string.gsub with a function replacement and a match limit (n < 0: all).
func gsubFunc(re *regexp.Regexp, s string, n int, repl func(groups []string) string) string {
	matches := re.FindAllStringSubmatchIndex(s, n)
	if matches == nil {
		return s
	}
	var out strings.Builder
	last := 0
	for _, m := range matches {
		groups := make([]string, len(m)/2)
		for g := range groups {
			if m[2*g] >= 0 {
				groups[g] = s[m[2*g]:m[2*g+1]]
			}
		}
		out.WriteString(s[last:m[0]])
		out.WriteString(repl(groups))
		last = m[1]
	}
	out.WriteString(s[last:])
	return out.String()
}

//pob2:src/Modules/Common.lua:722-728
func round(val float64, dec ...int) float64 {
	if len(dec) > 0 {
		p := math.Pow(10, float64(dec[0]))
		return math.Floor(val*p+0.5) / p
	}
	return math.Floor(val + 0.5)
}

//pob2:src/Modules/Common.lua:744-759
func roundSymmetric(val float64, dec ...int) float64 {
	factor := 1.0
	if len(dec) > 0 {
		factor = math.Pow(10, float64(dec[0]))
	}
	if val >= 0 {
		return math.Floor(val*factor+0.5) / factor
	}
	return math.Ceil(val*factor-0.5) / factor
}

//pob2:src/Modules/Common.lua:762-769
func alwaysPositiveRound(val float64) float64 {
	return floorSymmetric(val + 0.5)
}

//pob2:src/Modules/Common.lua:772-779
func floorSymmetric(val float64) float64 {
	return math.Trunc(val)
}

var sanitiseReplacements = strings.NewReplacer(
	"‐", "-",
	"‑", "-",
	"‒", "-",
	"–", "-",
	"—", "-",
	"―", "-",
	"−", "-",
	"• ", "",
	"•", "",
	"ä", "a",
	"ö", "o",
	"í", "i",
	"ó", "o",
)

//pob2:src/Modules/Common.lua:321-347
func sanitiseText(text string) string {
	found := false
	for i := 0; i < len(text); i++ {
		if text[i] >= 0x80 || text[i] == '<' {
			found = true
			break
		}
	}
	if !found {
		return text
	}
	text = stripBalanced(text, '<', '>')
	text = sanitiseReplacements.Replace(text)
	b := []byte(text)
	for i, c := range b {
		switch c {
		case 0x96, 0x97:
			b[i] = '-'
		case 0xe4:
			b[i] = 'a'
		case 0xf6:
			b[i] = 'o'
		default:
			if c >= 0x80 {
				b[i] = '?'
			}
		}
	}
	return string(b)
}

// stripBalanced is gsub("%bxy", "").
func stripBalanced(s string, open, closing byte) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != open {
			out.WriteByte(s[i])
			continue
		}
		depth, end := 0, -1
		for j := i; j < len(s); j++ {
			if s[j] == open {
				depth++
			} else if s[j] == closing {
				depth--
				if depth == 0 {
					end = j
					break
				}
			}
		}
		if end < 0 {
			out.WriteByte(s[i])
			continue
		}
		i = end
	}
	return out.String()
}

//pob2:src/Modules/Common.lua:1078-1084
func escapeGGGString(text string) string {
	line := gggTagRe.ReplaceAllString(text, "$1")
	line = gggLinkRe.ReplaceAllString(line, "$1")
	line = gggAltLinkRe.ReplaceAllString(line, "$1")
	return line
}

//pob2:src/Modules/ItemTools.lua:112-139
func applyValueScalar(line string, valueScalar, baseValueScalar *float64, numbers int, precision *int) string {
	scalar := 1.0
	if valueScalar != nil {
		scalar = *valueScalar
	}
	if scalar == 1 && (baseValueScalar == nil || *baseValueScalar == 1) {
		return line
	}
	if numbers == 0 {
		return line
	}
	if precision != nil {
		power := math.Pow(10, float64(*precision))
		return gsubFunc(valueScalarPrecRe, line, numbers, func(g []string) string {
			numVal, _ := luaToNumber(g[0])
			if baseValueScalar != nil {
				numVal = round(numVal**baseValueScalar*power) / power
			}
			numVal = math.Floor(numVal*scalar*power) / power
			return luaNum(numVal)
		})
	}
	return gsubFunc(valueScalarIntRe, line, numbers, func(g []string) string {
		numVal, _ := luaToNumber(g[1])
		if baseValueScalar != nil {
			numVal = round(numVal * *baseValueScalar)
		}
		numVal = math.Floor(numVal*scalar + 0.001)
		return luaNum(numVal) + g[2]
	})
}

//pob2:src/Modules/ItemTools.lua:143-156
func formatValue(value float64, baseValueScalar, valueScalar *float64, precision float64, displayPrecision *int, ifRequired bool) string {
	value = roundSymmetric(value * precision)
	if baseValueScalar != nil && *baseValueScalar != 1 {
		value = alwaysPositiveRound(value * *baseValueScalar)
	}
	if valueScalar != nil && *valueScalar != 1 {
		value = floorSymmetric(value * *valueScalar)
	}
	value /= precision
	if displayPrecision != nil {
		value = roundSymmetric(value, *displayPrecision)
	}
	if displayPrecision != nil && !ifRequired {
		return strconv.FormatFloat(value, 'f', *displayPrecision, 64)
	} else if displayPrecision != nil {
		return luaNum(value)
	}
	return luaNum(roundSymmetric(value, min(2, int(math.Floor(math.Log10(precision)+0.001)))))
}

//pob2:src/Modules/ItemTools.lua:158-168
func antonymFunc(num, word string) string {
	switch word {
	case "increased":
		return num + " reduced"
	case "reduced":
		return num + " increased"
	case "more":
		return num + " less"
	case "less":
		return num + " more"
	}
	return "-" + num + " " + word
}

//pob2:src/Modules/ItemTools.lua:170-172
func isZeroValueLine(line string) bool {
	return zeroLineStartRe.MatchString(line) ||
		(zeroLineMidRe.MatchString(line) && !zeroLineToRe.MatchString(line) && !zeroLinePctToRe.MatchString(line)) ||
		strings.Contains(line, " 0-0 ") || strings.Contains(line, " 0 to 0 ")
}

// ModScalability is one data.modScalability entry.
type ModScalability struct {
	IsScalable bool
	Formats    []string
}

// modScalabilityFor is data.modScalability[key]. data/poe2 carries no ModScalability table yet, so
// every line takes applyRange's fallback branch.
var modScalabilityFor = func(string) ([]ModScalability, bool) { return nil, false }

//pob2:src/Modules/ItemTools.lua:174-184
func replaceNthInstance(input, replacement string, n int) string {
	count := 0
	var out strings.Builder
	for i := 0; i < len(input); i++ {
		if input[i] == '#' {
			count++
			if count == n {
				out.WriteString(replacement)
				continue
			}
		}
		out.WriteByte(input[i])
	}
	return out.String()
}

func scalabilityKey(line string) string {
	return strings.ReplaceAll(line, "+#", "#")
}

//pob2:src/Modules/ItemTools.lua:186-221
func checkSubstitutionCombinations(i, numSubstitutions int, indices *[]int, line string, values []string) (string, []string, bool) {
	if len(*indices) == numSubstitutions {
		modifiedLine := line
		substituted := 0
		for _, idx := range *indices {
			modifiedLine = replaceNthInstance(modifiedLine, values[idx-1], idx-substituted)
			substituted++
		}
		if _, ok := modScalabilityFor(scalabilityKey(modifiedLine)); ok {
			used := map[int]bool{}
			for _, idx := range *indices {
				used[idx] = true
			}
			var remaining []string
			for idx, value := range values {
				if !used[idx+1] {
					remaining = append(remaining, value)
				}
			}
			return modifiedLine, remaining, true
		}
		return "", nil, false
	}
	for j := i; j <= len(values); j++ {
		*indices = append(*indices, j)
		if modifiedLine, remaining, ok := checkSubstitutionCombinations(j+1, numSubstitutions, indices, line, values); ok {
			return modifiedLine, remaining, true
		}
		*indices = (*indices)[:len(*indices)-1]
	}
	return "", nil, false
}

//pob2:block:src/Modules/ItemTools.lua:253-270
func findScalableLine(line string, values []string) (string, []string, bool) {
	for i := len(values); i >= 1; i-- {
		indices := []int{}
		if modifiedLine, remaining, ok := checkSubstitutionCombinations(1, i, &indices, line, values); ok {
			return modifiedLine, remaining, true
		}
	}
	if _, ok := modScalabilityFor(scalabilityKey(line)); ok {
		return line, values, true
	}
	return "", nil, false
}

//pob2:block:src/Modules/ItemTools.lua:279-372
func scalabilityFormat(formats []string) (float64, *int, bool) {
	precision := 0.0
	var displayPrecision *int
	ifRequired := false
	set := func(p float64, d int, req bool) {
		precision = p
		if d >= 0 {
			displayPrecision = &d
		}
		if req {
			ifRequired = true
		}
	}
	for _, format := range formats {
		switch format {
		case "divide_by_two_0dp":
			set(2, 0, true)
		case "divide_by_three":
			set(3, -1, false)
		case "divide_by_four":
			set(4, -1, false)
		case "divide_by_five":
			set(5, -1, false)
		case "divide_by_six":
			set(6, -1, false)
		case "divide_by_ten_0dp":
			set(10, 0, false)
		case "divide_by_ten_1dp":
			set(10, 1, false)
		case "divide_by_ten_1dp_if_required":
			set(10, 1, true)
		case "divide_by_twelve":
			set(12, -1, false)
		case "divide_by_fifteen_0dp":
			set(15, 0, false)
		case "divide_by_twenty":
			set(20, -1, false)
		case "divide_by_twenty_then_double_0dp":
			set(10, 0, false)
		case "divide_by_one_hundred", "divide_by_one_hundred_and_negate":
			set(100, -1, false)
		case "divide_by_one_hundred_0dp":
			set(100, 0, false)
		case "divide_by_one_hundred_1dp":
			set(100, 1, false)
		case "divide_by_one_hundred_2dp":
			set(100, 2, false)
		case "divide_by_one_hundred_2dp_if_required":
			set(100, 2, true)
		case "divide_by_one_thousand":
			set(1000, -1, false)
		case "divide_by_ten_thousand_1dp":
			set(10000, 1, false)
		case "per_minute_to_per_second":
			set(60, -1, false)
		case "per_minute_to_per_second_0dp":
			set(60, 0, false)
		case "per_minute_to_per_second_1dp":
			set(60, 1, false)
		case "per_minute_to_per_second_2dp":
			set(60, 2, false)
		case "per_minute_to_per_second_2dp_if_required":
			set(60, 2, true)
		case "milliseconds_to_seconds", "milliseconds_to_seconds_halved":
			set(1000, -1, false)
		case "milliseconds_to_seconds_0dp":
			set(1000, 0, false)
		case "milliseconds_to_seconds_1dp":
			set(1000, 1, false)
		case "milliseconds_to_seconds_2dp":
			set(1000, 2, false)
		case "locations_to_metres":
			set(10, 1, false)
		case "milliseconds_to_seconds_2dp_if_required":
			set(1000, 2, true)
		case "deciseconds_to_seconds":
			set(10, -1, false)
		}
	}
	return precision, displayPrecision, ifRequired
}

// applyRange applies a 0..1 roll (or one per range) to the line's "(x-y)" ranges. ranges, when
// non-nil, is the per-range table form of range.
//
//pob2:src/Modules/ItemTools.lua:228-436
func applyRange(line string, rangeVal float64, ranges []float64, valueScalar, baseValueScalar *float64, modSource ItemModSource) string {
	var values []string
	rangeIndex := 0
	strippedLine := gsubFunc(applyRangeRangeRe, line, -1, func(g []string) string {
		rangeIndex++
		valueRange := rangeVal
		if ranges != nil {
			valueRange = 0.5
			if rangeIndex <= len(ranges) {
				valueRange = ranges[rangeIndex-1]
			}
		}
		lo, _ := luaToNumber(g[2])
		hi, _ := luaToNumber(g[3])
		value := lo + valueRange*(hi-lo)
		if g[1] == "-" {
			value *= -1
		}
		if g[1] == "+" && value > 0 {
			return g[1] + luaNum(value)
		}
		return luaNum(value)
	})
	strippedLine = gsubFunc(applyRangeAntonymRe, strippedLine, -1, func(g []string) string { return antonymFunc(g[1], g[2]) })
	strippedLine = gsubFunc(applyRangeNumberRe, strippedLine, -1, func(g []string) string {
		values = append(values, g[0])
		return "#"
	})

	if scalableLine, scalableValues, ok := findScalableLine(strippedLine, values); ok {
		entries, _ := modScalabilityFor(scalabilityKey(scalableLine))
		for i, scalability := range entries {
			if i >= len(scalableValues) {
				break
			}
			precision, displayPrecision, ifRequired := scalabilityFormat(scalability.Formats)
			if precision == 0 {
				precision = 1
			}
			v, _ := luaToNumber(scalableValues[i])
			one := 1.0
			if scalability.IsScalable && ((baseValueScalar != nil && *baseValueScalar != 1) || (valueScalar != nil && *valueScalar != 1)) {
				scalableValues[i] = formatValue(v, baseValueScalar, valueScalar, precision, displayPrecision, ifRequired)
			} else {
				scalableValues[i] = formatValue(v, &one, &one, precision, displayPrecision, ifRequired)
			}
		}
		for _, replacement := range scalableValues {
			scalableLine = strings.Replace(scalableLine, "#", replacement, 1)
		}
		return scalableLine
	}

	// The fallback ranges by the single range value, as the Lua does.
	precisionSame := true
	testLine := line
	if strings.Contains(line, "-") {
		testLine = gsubFunc(applyRangePlusRangeRe, line, -1, func(g []string) string {
			lo, _ := luaToNumber(g[2])
			hi, _ := luaToNumber(g[3])
			maxPrecision := lo + rangeVal*(hi-lo)
			minPrecision := math.Floor(maxPrecision + 0.5)
			if minPrecision != maxPrecision {
				precisionSame = false
			}
			if minPrecision < 0 {
				return luaNum(minPrecision)
			}
			return g[1] + luaNum(minPrecision)
		})
		testLine = gsubFunc(applyRangeIntAntonyRe, testLine, -1, func(g []string) string { return antonymFunc(g[1], g[2]) })
	}
	if precisionSame && (valueScalar == nil || *valueScalar == 1) && (baseValueScalar == nil || *baseValueScalar == 1) {
		return testLine
	}

	var precision *int
	if modList, extra := modSource(testLine); modList != nil && extra == "" {
		for _, m := range modList {
			sub := nestedMod(m)
			if sub.Value() != nil && sub.Value().Type() == "float" {
				if p, ok := highPrecisionMods[sub.Name()][sub.Type()]; ok {
					precision = &p
				}
			}
		}
	}
	if precision == nil && applyRangeDecimalRe.MatchString(line) {
		p := defaultHighPrecision
		precision = &p
	}
	numbers := 0
	line = gsubFunc(applyRangePlusRangeRe, line, -1, func(g []string) string {
		numbers++
		power := 1.0
		if precision != nil {
			power = math.Pow(10, float64(*precision))
		}
		lo, _ := luaToNumber(g[2])
		hi, _ := luaToNumber(g[3])
		numVal := math.Floor((lo+rangeVal*(hi-lo))*power+0.5) / power
		if numVal < 0 {
			return luaNum(numVal)
		}
		return g[1] + luaNum(numVal)
	})
	line = gsubFunc(applyRangeIntAntonyRe, line, -1, func(g []string) string { return antonymFunc(g[1], g[2]) })
	if numbers == 0 && applyRangeScalableRe.MatchString(line) {
		numbers = 1
	}
	return applyValueScalar(line, valueScalar, baseValueScalar, numbers, precision)
}

//pob2:src/Modules/Data.lua:480-480
const defaultHighPrecision = 1

// RangedLine is the text itemLib.formatModLine shows for modLine outside dbMode: the line with its
// range and value scalar applied.
//
//pob2:block:src/Modules/ItemTools.lua:439-440
func (i *Item) RangedLine(modLine *ModLine) string {
	valueScalar := modLine.ValueScalar
	if modLine.DisplayValueScalar != nil {
		v := 1.0
		if modLine.ValueScalar != nil {
			v = *modLine.ValueScalar
		}
		valueScalar = floatp(v * *modLine.DisplayValueScalar)
	}
	if modLine.Range == nil && modLine.DisplayValueScalar == nil {
		return modLine.Line
	}
	rangeVal := defaultItemAffixQuality
	if modLine.Range != nil {
		rangeVal = *modLine.Range
	}
	return applyRange(modLine.Line, rangeVal, nil, valueScalar, modLine.CorruptedRange, i.modSource)
}
