package raw

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/loader"
)

// tradeStatsFile is go-pob-data's copy of https://www.pathofexile.com/api/trade/data/stats, as the trade site
// served it.
const tradeStatsFile = "api/trade/data/stats.json"

// TradeStatCategory is one category of the trade site's stats list, such as explicit or implicit.
type TradeStatCategory struct {
	ID      string           `json:"id"`
	Label   string           `json:"label"`
	Entries []TradeStatEntry `json:"entries"`
}

type TradeStatEntry struct {
	Option *TradeStatOptions `json:"option,omitempty"`
	ID     string            `json:"id"`
	Text   string            `json:"text"`
	Type   string            `json:"type"`
}

// TradeStatOptions are the choices of a stat the trade site searches by a dropdown.
type TradeStatOptions struct {
	Options []TradeStatOption `json:"options"`
}

// TradeStatOption is one choice: ID is a number or a string, as the site serves it.
type TradeStatOption struct {
	ID   any    `json:"id"`
	Text string `json:"text"`
}

var loadTradeStats = sync.OnceValues(func() ([]TradeStatCategory, error) {
	b, err := loader.LoadFile(context.Background(), LatestVersion, tradeStatsFile, lazyLoader())
	if err != nil {
		return nil, fmt.Errorf("trade stats: %w", err)
	}
	return decodeTradeStats(b)
})

// TradeStats is Data/TradeSiteStats.lua: the trade site's stats list, an error when go-pob-data does not publish it.
func TradeStats() ([]TradeStatCategory, error) {
	return loadTradeStats()
}

// decodeTradeStats is the response as TradeQueryGenerator writes it to TradeSiteStats.lua: each category's entries
// sorted by text then id, and every text's line breaks turned into spaces by its stringify.
//
//pob:block:src/Classes/TradeQueryGenerator.lua:393-414
//pob:block:src/Modules/Utils.lua:25-25
func decodeTradeStats(b []byte) ([]TradeStatCategory, error) {
	var body struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Result []TradeStatCategory `json:"result"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, fmt.Errorf("trade stats: %w", err)
	}
	if body.Error != nil {
		return nil, fmt.Errorf("error received from api/trade/data/stats: %s", body.Error.Message)
	}
	newlines := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")
	for _, cat := range body.Result {
		sort.SliceStable(cat.Entries, func(i, j int) bool {
			a, b := cat.Entries[i], cat.Entries[j]
			if a.Text == b.Text {
				return a.ID < b.ID
			}
			return a.Text < b.Text
		})
		for i := range cat.Entries {
			e := &cat.Entries[i]
			e.Text = newlines.Replace(e.Text)
			if e.Option != nil {
				for j := range e.Option.Options {
					e.Option.Options[j].Text = newlines.Replace(e.Option.Options[j].Text)
				}
			}
		}
	}
	return body.Result, nil
}
