package poe2

import (
	"context"
	"fmt"
	"sync"

	"github.com/Vilsol/go-pob-data/poe2/pobdata"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/storage"
)

// itemTablesVersion is the go-pob-data version whose pob/ assets hold the item tables.
const itemTablesVersion = "0.5"

type itemTableSet struct {
	bases          []ItemBase
	mods           []ItemMod
	exclusive      []ExclusiveMod
	modScalability map[string][]ModScalabilityEntry
}

var loadItemTables = sync.OnceValues(func() (*itemTableSet, error) {
	ctx := context.Background()
	cache := assetcache.AssetLoaderWrapper{Storage: storage.Get()}
	bases, err := pobdata.Load[ItemBase](ctx, itemTablesVersion, itemBasesAsset, cache)
	if err != nil {
		return nil, fmt.Errorf("item bases: %w", err)
	}
	mods, err := pobdata.Load[ItemMod](ctx, itemTablesVersion, itemModsAsset, cache)
	if err != nil {
		return nil, fmt.Errorf("item mods: %w", err)
	}
	exclusive, err := pobdata.Load[ExclusiveMod](ctx, itemTablesVersion, exclusiveModsAsset, cache)
	if err != nil {
		return nil, fmt.Errorf("exclusive mods: %w", err)
	}
	scalability, err := pobdata.Load[pobdata.ModScalability](ctx, itemTablesVersion, modScalabilityAsset, cache)
	if err != nil {
		return nil, fmt.Errorf("mod scalability: %w", err)
	}
	return &itemTableSet{
		bases:          bases,
		mods:           mods,
		exclusive:      exclusive,
		modScalability: modScalabilityByLine(scalability),
	}, nil
})

// LoadItemTables loads data.itemBases, data.itemMods and data.modScalability from go-pob-data. Every
// reader of them loads them on first use and panics with this error, so callers that can return one
// call this first.
func LoadItemTables() error {
	_, err := loadItemTables()
	return err
}

func itemTables() *itemTableSet {
	tables, err := loadItemTables()
	if err != nil {
		panic(fmt.Errorf("PoE2 item tables: %w", err))
	}
	return tables
}

// ItemBases is data.itemBases, sorted by name.
func ItemBases() []ItemBase {
	return itemTables().bases
}

// ExclusiveMods is data.itemMods.Exclusive's lines and stat orders, sorted by id.
func ExclusiveMods() []ExclusiveMod {
	return itemTables().exclusive
}
