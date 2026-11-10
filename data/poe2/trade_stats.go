package poe2

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/loader"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/storage"
)

// tradeStatsFile is go-pob-data's copy of https://www.pathofexile.com/api/trade2/data/stats, as the trade site
// served it.
const tradeStatsFile = "api/trade2/data/stats.json"

// TradeStatCategory is one category of the trade site's stats list, such as explicit or implicit.
type TradeStatCategory struct {
	ID      string           `json:"id"`
	Label   string           `json:"label"`
	Entries []TradeStatEntry `json:"entries"`
}

type TradeStatEntry struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Type string `json:"type"`
}

var loadTradeStats = sync.OnceValues(func() ([]TradeStatCategory, error) {
	ctx := context.Background()
	b, err := loader.LoadFile(ctx, itemTablesVersion, tradeStatsFile, assetcache.AssetLoaderWrapper{Storage: storage.Get()})
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
//pob2:block:src/Classes/TradeQueryGenerator.lua:400-414
//pob2:block:src/Modules/Utils.lua:25-25
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
		return nil, fmt.Errorf("error received from api/trade2/data/stats: %s", body.Error.Message)
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
			cat.Entries[i].Text = newlines.Replace(cat.Entries[i].Text)
		}
	}
	return body.Result, nil
}
