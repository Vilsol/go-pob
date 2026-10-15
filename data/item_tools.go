package data

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	decimalRe        = regexp.MustCompile(`\d+\.?\d*`)
	integerRe        = regexp.MustCompile(`(\d+)([^.])`)
	pairRangeRe      = regexp.MustCompile(`\((\d+)-(\d+) to (\d+)-(\d+)\)`)
	toRangeRe        = regexp.MustCompile(`(\+?)\((-?\d+) to (\d+)\)`)
	intRangeRe       = regexp.MustCompile(`(\+?)\((-?\d+)-(\d+)\)`)
	decRangeRe       = regexp.MustCompile(`\((\d+\.?\d*)-(\d+\.?\d*)\)`)
	rangeRe          = regexp.MustCompile(`(\+?)\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)
	negIncreasedRe   = regexp.MustCompile(`-(\d+%) increased`)
	negReducedRe     = regexp.MustCompile(`-(\d+%) reduced`)
	negMoreRe        = regexp.MustCompile(`-(\d+%) more`)
	negLessRe        = regexp.MustCompile(`-(\d+%) less`)
	scalableNumberRe = regexp.MustCompile(`\d+\.?\d*%? `)
	realDecimalRe    = regexp.MustCompile(`\d+\.\d*`)
)

const defaultHighPrecision = 1

func luaNum(v float64) string {
	return strconv.FormatFloat(v, 'g', 14, 64)
}

func replaceLimit(re *regexp.Regexp, s string, n int, repl func(groups []string) string) string {
	if n == 0 {
		return s
	}

	matches := re.FindAllStringSubmatchIndex(s, -1)
	if matches == nil {
		return s
	}

	var out strings.Builder
	last := 0
	for i, m := range matches {
		if n > 0 && i >= n {
			break
		}

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

//pob:src/Modules/ItemTools.lua:26-43
func ApplyValueScalar(line string, valueScalar, baseValueScalar *float64, numbers int, precision *int) string {
	scalar := 1.0
	if valueScalar != nil {
		scalar = *valueScalar
	}
	if scalar == 1 && (baseValueScalar == nil || *baseValueScalar == 1) {
		return line
	}

	power := 1.0
	if precision != nil {
		power = math.Pow(10, float64(*precision))
	}
	scaleBase := func(v float64) float64 {
		if baseValueScalar == nil || *baseValueScalar == 1 {
			return v
		}
		return math.Floor(float64(v**baseValueScalar*power)+0.5) / power
	}

	if precision != nil {
		return replaceLimit(decimalRe, line, numbers, func(g []string) string {
			num, _ := strconv.ParseFloat(g[0], 64)
			return luaNum(math.Floor(scaleBase(num)*scalar*power) / power)
		})
	}

	return replaceLimit(integerRe, line, numbers, func(g []string) string {
		num, _ := strconv.ParseFloat(g[1], 64)
		return luaNum(math.Floor(float64(scaleBase(num)*scalar)+0.001)) + g[2]
	})
}

var (
	lineRangeMu    sync.RWMutex
	lineRangeCache = make(map[string]lineRange)
)

type lineRange struct {
	min, max string
	ok       bool
}

//pob:src/Modules/ItemTools.lua:45-60
func GetLineRangeMinMax(line string) (string, string, bool) {
	lineRangeMu.RLock()
	cached, hit := lineRangeCache[line]
	lineRangeMu.RUnlock()
	if hit {
		return cached.min, cached.max, cached.ok
	}

	rMin, rMax, rOK := getLineRangeMinMaxUncached(line)

	lineRangeMu.Lock()
	lineRangeCache[line] = lineRange{min: rMin, max: rMax, ok: rOK}
	lineRangeMu.Unlock()

	return rMin, rMax, rOK
}

func getLineRangeMinMaxUncached(line string) (string, string, bool) {
	var rangeMin, rangeMax string
	var ok bool

	line = pairRangeRe.ReplaceAllString(line, "($1-$2) to ($3-$4)")
	line = toRangeRe.ReplaceAllString(line, "$1($2-$3)")

	line = replaceLimit(intRangeRe, line, -1, func(g []string) string {
		rangeMin, rangeMax, ok = g[2], g[3], true
		return ""
	})

	replaceLimit(decRangeRe, line, -1, func(g []string) string {
		rangeMin, rangeMax, ok = g[1], g[2], true
		return ""
	})

	return rangeMin, rangeMax, ok
}

var (
	applyRangeMu    sync.RWMutex
	applyRangeCache = make(map[applyRangeKey]string)
)

type applyRangeKey struct {
	line          string
	rangeVal      float64
	scalar        float64
	hasScale      bool
	baseScalar    float64
	hasBaseScalar bool
}

//pob:src/Modules/ItemTools.lua:62-115
func ApplyRange(line string, rangeVal float64, valueScalar, baseValueScalar *float64) string {
	key := applyRangeKey{line: line, rangeVal: rangeVal}
	if valueScalar != nil {
		key.scalar, key.hasScale = *valueScalar, true
	}
	if baseValueScalar != nil {
		key.baseScalar, key.hasBaseScalar = *baseValueScalar, true
	}

	applyRangeMu.RLock()
	cached, ok := applyRangeCache[key]
	applyRangeMu.RUnlock()
	if ok {
		return cached
	}

	out := applyRangeUncached(line, rangeVal, valueScalar, baseValueScalar)

	applyRangeMu.Lock()
	applyRangeCache[key] = out
	applyRangeMu.Unlock()

	return out
}

func applyRangeUncached(line string, rangeVal float64, valueScalar, baseValueScalar *float64) string {
	if out, ok := applyRangeScalability(line, rangeVal, valueScalar, baseValueScalar); ok {
		return out
	}

	numbers := 0

	var precision *int
	if realDecimalRe.MatchString(line) {
		p := defaultHighPrecision
		precision = &p
	}

	line = replaceLimit(rangeRe, line, -1, func(g []string) string {
		plus := g[1]
		rangeLow, _ := strconv.ParseFloat(g[2], 64)
		rangeHigh, _ := strconv.ParseFloat(g[3], 64)

		numbers++
		power := 1.0
		if precision != nil {
			power = math.Pow(10, float64(*precision))
		}
		numVal := math.Floor(float64((rangeLow+float64(rangeVal*(rangeHigh-rangeLow)))*power)+0.5) / power
		if numVal < 0 && plus == "+" {
			plus = ""
		}

		return plus + luaNum(numVal)
	})

	line = negIncreasedRe.ReplaceAllString(line, "$1 reduced")
	line = negReducedRe.ReplaceAllString(line, "$1 increased")
	line = negMoreRe.ReplaceAllString(line, "$1 less")
	line = negLessRe.ReplaceAllString(line, "$1 more")

	if numbers == 0 && scalableNumberRe.MatchString(line) {
		numbers = 1
	}

	return ApplyValueScalar(line, valueScalar, baseValueScalar, numbers, precision)
}

const luaSpace = " \t\n\v\f\r"

func stripBalancedAngles(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '<' {
			out.WriteByte(s[i])
			continue
		}

		depth := 0
		end := -1
		for j := i; j < len(s); j++ {
			switch s[j] {
			case '<':
				depth++
			case '>':
				depth--
				if depth == 0 {
					end = j
				}
			}
			if end >= 0 {
				break
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

func sanitiseNeeded(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] >= 0x80 || text[i] == '<' {
			return true
		}
	}
	return false
}

var sanitiseReplacements = []struct{ from, to string }{
	{"‐", "-"},
	{"‑", "-"},
	{"‒", "-"},
	{"–", "-"},
	{"—", "-"},
	{"―", "-"},
	{"−", "-"},
	{"ä", "a"},
	{"ö", "o"},
	// single-byte: Windows-1252 and similar
	{"\x96", "-"},
	{"\x97", "-"},
	{"\xe4", "a"},
	{"\xf6", "o"},
}

//pob:src/Modules/Common.lua:303-329
func SanitiseItemText(text string) string {
	if !sanitiseNeeded(text) {
		return text
	}

	text = stripBalancedAngles(text)

	for _, r := range sanitiseReplacements {
		text = strings.ReplaceAll(text, r.from, r.to)
	}

	b := []byte(text)
	for i, c := range b {
		if c >= 0x80 {
			b[i] = '?'
		}
	}

	return string(b)
}
