package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

type scalability struct {
	IsScalable bool     `json:"isScalable"`
	Formats    []string `json:"formats,omitempty"`
}

var (
	scalabilityPlaceholderRe = regexp.MustCompile(`[+-]?\{[^{}]*\}`)
	scalabilityDigitRe       = regexp.MustCompile(`\d`)
)

//pob:src/Export/statdesc.lua:463-505
func (e *exportDescriber) describeScalability() (map[string][]scalability, error) {
	isScalable := map[string]bool{}
	for _, s := range poe.Stats {
		isScalable[s.ID] = s.IsScalable
	}
	keys := make([]string, 0, len(e.descriptors))
	for stat := range e.descriptors {
		keys = append(keys, stat)
	}
	sort.Strings(keys)

	out := map[string][]scalability{}
	for _, stat := range keys {
		statDescription := e.descriptors[stat]
		if statDescription.stats == nil {
			continue
		}
		var scalable []bool
		for _, s := range statDescription.stats {
			v, ok := isScalable[s]
			if !ok {
				return nil, fmt.Errorf("%s is not a Stats row", s)
			}
			scalable = append(scalable, v)
		}
		for _, wordings := range statDescription.variants {
			wordingFormats := map[int][]string{}
			for _, format := range wordings.Specs {
				if v, ok := format.V.(float64); ok {
					wordingFormats[int(v)] = append(wordingFormats[int(v)], format.K)
				}
			}
			var inOrder []scalability
			strippedLine := scalabilityPlaceholderRe.ReplaceAllStringFunc(wordings.Text, func(num string) string {
				statNum := 1
				if d := scalabilityDigitRe.FindString(num); d != "" {
					statNum = int(d[0]-'0') + 1
				}
				entry := scalability{Formats: wordingFormats[statNum]}
				if statNum <= len(scalable) {
					entry.IsScalable = scalable[statNum-1]
				}
				inOrder = append(inOrder, entry)
				return "#"
			})
			if strings.HasPrefix(strippedLine, "DNT") {
				continue
			}
			strippedLine = strings.ReplaceAll(strippedLine, `\n`, "\n")
			prior, ok := out[strippedLine]
			if !ok {
				out[strippedLine] = inOrder
				continue
			}
			for j := range prior {
				if len(prior[j].Formats) > len(wordingFormats[j+1]) {
					if j < len(inOrder) {
						prior[j] = inOrder[j]
					} else {
						return nil, fmt.Errorf("%q: placeholder %d has no counterpart", strippedLine, j+1)
					}
				}
			}
		}
	}
	return out, nil
}

//pob:block:src/Export/Scripts/modScalability.lua:1-31
func dumpModScalability() (any, error) {
	ctx, cache := gameDataContext(), assetCache()
	if err := poe.InitializeStats(ctx, raw.LatestVersion, cache); err != nil {
		return nil, fmt.Errorf("Stats: %w", err)
	}
	describer, err := loadStatFile("stat_descriptions")
	if err != nil {
		return nil, err
	}
	rows, err := describer.describeScalability()
	if err != nil {
		return nil, err
	}
	return map[string]any{"pobVersion": "go-pob-data " + raw.LatestVersion, "rows": rows}, nil
}
