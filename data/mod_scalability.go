package data

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data/raw"
)

var (
	scalabilityRangeRe   = regexp.MustCompile(`([+-]?)\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)
	scalabilityAntonymRe = regexp.MustCompile(`-(\d+\.?\d*%) ([a-zA-Z]+)`)
	scalabilityNumberRe  = regexp.MustCompile(`-?\d+\.?\d*`)
	hashRe               = regexp.MustCompile(`#`)
)

var antonyms = map[string]string{
	"increased": "reduced",
	"reduced":   "increased",
	"more":      "less",
	"less":      "more",
}

func applyAntonym(num, word string) string {
	if antonym, ok := antonyms[word]; ok {
		return num + " " + antonym
	}
	return "-" + num + " " + word
}

func roundSymmetric(val float64) float64 {
	if val >= 0 {
		return math.Floor(float64(val) + 0.5)
	}
	return math.Ceil(float64(val) - 0.5)
}

func roundSymmetricDec(val float64, dec int) float64 {
	factor := math.Pow(10, float64(dec))
	if val >= 0 {
		return math.Floor(float64(val*factor)+0.5) / factor
	}
	return math.Ceil(float64(val*factor)-0.5) / factor
}

func floorSymmetric(val float64) float64 {
	return math.Trunc(val)
}

func alwaysPositiveRound(val float64) float64 {
	return floorSymmetric(float64(val) + 0.5)
}

//pob:src/Modules/ItemTools.lua:61-75
func formatValue(value float64, baseValueScalar, valueScalar *float64, precision float64,
	displayPrecision *int, ifRequired bool,
) string {
	value = roundSymmetric(value * precision)
	if baseValueScalar != nil && *baseValueScalar != 1 {
		value = alwaysPositiveRound(value * *baseValueScalar)
	}
	if valueScalar != nil && *valueScalar != 1 {
		value = floorSymmetric(value * *valueScalar)
	}
	value /= precision

	if displayPrecision != nil {
		value = roundSymmetricDec(value, *displayPrecision)
		if !ifRequired {
			return fmt.Sprintf("%.*f", *displayPrecision, value)
		}
		return luaNum(value)
	}

	dec := int(math.Floor(float64(math.Log10(precision)) + 0.001))
	if dec > 2 {
		dec = 2
	}
	return luaNum(roundSymmetricDec(value, dec))
}

//pob:block:src/Modules/ItemTools.lua:191-283
func precisionForFormats(formats []string) (*float64, *int, bool) {
	var precision *float64
	var displayPrecision *int
	var ifRequired bool
	setP := func(v float64) { precision = &v }
	setD := func(v int) { displayPrecision = &v }
	for _, format := range formats {
		switch format {
		case "divide_by_two_0dp":
			setP(2)
			setD(0)
			ifRequired = true
		case "divide_by_three":
			setP(3)
		case "divide_by_four":
			setP(4)
		case "divide_by_five":
			setP(5)
		case "divide_by_six":
			setP(6)
		case "divide_by_ten_0dp":
			setP(10)
			setD(0)
		case "divide_by_ten_1dp":
			setP(10)
			setD(1)
		case "divide_by_ten_1dp_if_required":
			setP(10)
			setD(1)
			ifRequired = true
		case "divide_by_twelve":
			setP(12)
		case "divide_by_fifteen_0dp":
			setP(15)
			setD(0)
		case "divide_by_twenty":
			setP(20)
		case "divide_by_twenty_then_double_0dp":
			setP(10)
			setD(0)
		case "divide_by_one_hundred", "divide_by_one_hundred_and_negate":
			setP(100)
		case "divide_by_one_hundred_0dp":
			setP(100)
			setD(0)
		case "divide_by_one_hundred_1dp":
			setP(100)
			setD(1)
		case "divide_by_one_hundred_2dp":
			setP(100)
			setD(2)
		case "divide_by_one_hundred_2dp_if_required":
			setP(100)
			setD(2)
			ifRequired = true
		case "divide_by_one_thousand":
			setP(1000)
		case "divide_by_ten_thousand_1dp":
			setP(10000)
			setD(1)
		case "per_minute_to_per_second":
			setP(60)
		case "per_minute_to_per_second_0dp":
			setP(60)
			setD(0)
		case "per_minute_to_per_second_1dp":
			setP(60)
			setD(1)
		case "per_minute_to_per_second_2dp":
			setP(60)
			setD(2)
		case "per_minute_to_per_second_2dp_if_required":
			setP(60)
			setD(2)
			ifRequired = true
		case "milliseconds_to_seconds", "milliseconds_to_seconds_halved":
			setP(1000)
		case "milliseconds_to_seconds_0dp":
			setP(1000)
			setD(0)
		case "milliseconds_to_seconds_1dp":
			setP(1000)
			setD(1)
		case "milliseconds_to_seconds_2dp":
			setP(1000)
			setD(2)
		case "locations_to_metres":
			setP(10)
			setD(1)
		case "milliseconds_to_seconds_2dp_if_required":
			setP(1000)
			setD(2)
			ifRequired = true
		case "deciseconds_to_seconds":
			setP(10)
		}
	}
	return precision, displayPrecision, ifRequired
}

func scalabilityKey(line string) string {
	return strings.ReplaceAll(line, "+#", "#")
}

var scalabilityIndex = sync.OnceValue(func() map[string][]string {
	idx := make(map[string][]string, len(raw.ModScalability))
	for key := range raw.ModScalability {
		b := key
		if strings.ContainsAny(key, "0123456789") {
			b = scalabilityNumberRe.ReplaceAllString(key, "#")
		}
		b = scalabilityKey(b)
		idx[b] = append(idx[b], key)
	}
	return idx
})

func alignScalabilityKey(parts []string, values []string, key string) []int {
	var subs []int
	pos := 0
	for i := range values {
		lit := parts[i]
		keepLit := lit
		if strings.HasSuffix(lit, "+") {
			keepLit = lit[:len(lit)-1]
		}
		switch {
		case strings.HasPrefix(key[pos:], keepLit+"#"):
			pos += len(keepLit) + 1
		case strings.HasPrefix(key[pos:], lit+values[i]):
			pos += len(lit) + len(values[i])
			subs = append(subs, i)
		default:
			return nil
		}
	}
	if key[pos:] != parts[len(values)] {
		return nil
	}
	if subs == nil {
		subs = []int{}
	}
	return subs
}

func substituteScalability(parts []string, values []string, subs []int) string {
	subbed := make(map[int]bool, len(subs))
	for _, i := range subs {
		subbed[i] = true
	}
	var b strings.Builder
	for i := range values {
		b.WriteString(parts[i])
		if subbed[i] {
			b.WriteString(values[i])
		} else {
			b.WriteString("#")
		}
	}
	b.WriteString(parts[len(values)])
	return b.String()
}

//pob:block:src/Modules/ItemTools.lua:117-182
func findScalableLine(line string, values []string) (string, []string, bool) {
	candidates := scalabilityIndex()[scalabilityKey(line)]
	if len(candidates) == 0 {
		return "", nil, false
	}

	parts := strings.Split(line, "#")
	if len(parts) != len(values)+1 {
		return "", nil, false
	}

	best := []int(nil)
	for _, key := range candidates {
		subs := alignScalabilityKey(parts, values, key)
		if subs == nil {
			continue
		}
		if best == nil || moreSubstituted(subs, best) {
			best = subs
		}
	}
	if best == nil {
		return "", nil, false
	}

	subbed := make(map[int]bool, len(best))
	for _, i := range best {
		subbed[i] = true
	}
	var remaining []string
	for i, v := range values {
		if !subbed[i] {
			remaining = append(remaining, v)
		}
	}
	return substituteScalability(parts, values, best), remaining, true
}

func moreSubstituted(a, b []int) bool {
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

//pob:block:src/Modules/ItemTools.lua:94-108
func stripLineForScalability(line string, rangeVal float64) (string, []string) {
	stripped := replaceLimit(scalabilityRangeRe, line, -1, func(g []string) string {
		sign := g[1]
		rangeLow, _ := strconv.ParseFloat(g[2], 64)
		rangeHigh, _ := strconv.ParseFloat(g[3], 64)
		value := rangeLow + float64(rangeVal*(rangeHigh-rangeLow))
		if sign == "-" {
			value *= -1
		}
		if sign == "+" && value > 0 {
			return sign + luaNum(value)
		}
		return luaNum(value)
	})

	stripped = replaceLimit(scalabilityAntonymRe, stripped, -1, func(g []string) string {
		return applyAntonym(g[1], g[2])
	})

	var values []string
	stripped = replaceLimit(scalabilityNumberRe, stripped, -1, func(g []string) string {
		values = append(values, g[0])
		return "#"
	})

	return stripped, values
}

//pob:block:src/Modules/ItemTools.lua:184-296
func applyRangeScalability(line string, rangeVal float64, valueScalar, baseValueScalar *float64) (string, bool) {
	stripped, values := stripLineForScalability(line, rangeVal)

	scalableLine, scalableValues, ok := findScalableLine(stripped, values)
	if !ok {
		return "", false
	}

	one := 1.0
	for i, entry := range raw.ModScalability[scalabilityKey(scalableLine)] {
		if i >= len(scalableValues) {
			break
		}
		precision, displayPrecision, ifRequired := precisionForFormats(entry.Formats)
		p := 1.0
		if precision != nil {
			p = *precision
		}
		value, err := strconv.ParseFloat(scalableValues[i], 64)
		if err != nil {
			continue
		}
		if entry.IsScalable && ((baseValueScalar != nil && *baseValueScalar != 1) ||
			(valueScalar != nil && *valueScalar != 1)) {
			scalableValues[i] = formatValue(value, baseValueScalar, valueScalar, p, displayPrecision, ifRequired)
		} else {
			scalableValues[i] = formatValue(value, &one, &one, p, displayPrecision, ifRequired)
		}
	}

	for _, replacement := range scalableValues {
		scalableLine = replaceLimit(hashRe, scalableLine, 1, func([]string) string { return replacement })
	}
	return scalableLine, true
}
