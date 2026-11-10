package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func BuySimilarExports(r bind.Registry) {
	r.Type(BuySimilarView{}, bind.Plain())
	r.Type(calculator.BuySimilarDefence{}, bind.Plain())
	r.Type(calculator.BuySimilarModRow{}, bind.Plain())
	r.Type(calculator.BuySimilarOptions{}, bind.Plain())
	r.Type(calculator.BuySimilarDefenceFilter{}, bind.Plain())
	r.Type(calculator.BuySimilarModFilter{}, bind.Plain())

	// async: the mod rows load the trade site's stats list and the stat descriptions on first use
	r.Func(BuySimilar, bind.AsPromise())
	r.Func(BuySimilarURL, bind.AsPromise())
}

// BuySimilarView is the Buy Similar popup for an item, as the Compare tab opens it for the compared build's item:
// whether it searches by unique name, the name it shows, the Category label, the base the base box names, the defence
// rows and the mod rows. There are no mod rows when go-pob-data does not publish the trade site's stats list.
type BuySimilarView struct {
	Name     string
	BaseName string
	Category string
	Defences []calculator.BuySimilarDefence
	Mods     []calculator.BuySimilarModRow
	Unique   bool
}

// buySimilarModEntries is the popup's mod entries for item; none when the trade data cannot be read.
func buySimilarModEntries(item *calculator.Item) []*calculator.BuySimilarModEntry {
	if _, err := raw.TradeStats(); err != nil {
		return nil
	}
	if _, err := raw.ItemStatDescriptions(); err != nil {
		return nil
	}
	return calculator.BuySimilarAddModEntries(item)
}

// BuySimilar is the popup for the item text in slotName, read as build reads it.
//
//pob:block:src/Classes/CompareBuySimilar.lua:285-333
//pob:block:src/Classes/CompareBuySimilar.lua:418-431
func BuySimilar(build *pob.PathOfBuilding, text string, slotName string) (*BuySimilarView, error) {
	item, err := buildItem(build, text)
	if err != nil {
		return nil, err
	}
	return &BuySimilarView{
		Unique:   item.Rarity == calculator.RarityUnique || item.Rarity == calculator.RarityRelic,
		Name:     item.Name,
		BaseName: item.BaseName,
		Category: calculator.TradeCategoryLabel(item),
		Defences: calculator.BuySimilarDefences(item),
		Mods:     calculator.BuySimilarModRows(buySimilarModEntries(item)),
	}, nil
}

// BuySimilarURL is the Open URL button's trade search for the item text in slotName with the popup's selections.
func BuySimilarURL(build *pob.PathOfBuilding, text string, slotName string, o calculator.BuySimilarOptions) (string, error) {
	item, err := buildItem(build, text)
	if err != nil {
		return "", err
	}
	return calculator.BuySimilarURL(item, slotName, buySimilarModEntries(item), o), nil
}
