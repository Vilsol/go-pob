package raw

import (
	"context"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/poe"
)

// itemArtTables load on the first art lookup rather than with the other deferred tables, and are not
// prefetched: only the icons read them.
var itemArtTables = []table{
	{"ItemVisualIdentity", poe.InitializeItemVisualIdentities},
	{"Words", poe.InitializeWords},
	{"UniqueStashLayout", poe.InitializeUniqueStashLayouts},
}

var (
	itemArtMutex  sync.Mutex
	itemArtLoaded bool
)

// ensureItemArt loads itemArtTables once the base types are, and tries again on the next call when a
// load fails (offline).
func ensureItemArt() bool {
	itemArtMutex.Lock()
	defer itemArtMutex.Unlock()
	if itemArtLoaded {
		return true
	}
	if len(poe.BaseItemTypes) == 0 {
		return false
	}
	ctx := DataContext(context.Background())
	for _, t := range itemArtTables {
		if err := t.Load(ctx, LatestVersion, lazyLoader()); err != nil {
			return false
		}
	}
	buildArtLookups()
	itemArtLoaded = true
	return true
}

var (
	baseTypeByName map[string]*poe.BaseItemType
	gemGameIDByArt map[string]string
)

func buildArtLookups() {
	baseTypeByName = make(map[string]*poe.BaseItemType, len(poe.BaseItemTypes))
	for _, b := range poe.BaseItemTypes {
		if strings.HasPrefix(b.ID, "Metadata/Items/Gems/") {
			continue
		}
		if _, ok := baseTypeByName[b.Name]; !ok {
			baseTypeByName[b.Name] = b
		}
	}
	gemGameIDByArt = make(map[string]string, len(Gems)+len(GemVariants))
	add := func(name, gameID string) {
		key := strings.ToLower(name)
		if prev, ok := gemGameIDByArt[key]; name != "" && gameID != "" && (!ok || gameID < prev) {
			gemGameIDByArt[key] = gameID
		}
	}
	for gameID, row := range Gems {
		add(row.Name, gameID)
	}
	for _, row := range GemVariants {
		add(row.Name, GemGameIDForGrantedEffectID[row.GrantedEffectID])
	}
}

func artPath(b *poe.BaseItemType) string {
	if b == nil {
		return ""
	}
	if v := b.GetItemVisualIdentity(); v != nil {
		return v.ArtPath()
	}
	return ""
}

// ItemArtPath is the game art path of an item: a unique's own art when name is a unique's, otherwise
// its base type's. Empty when neither is known.
func ItemArtPath(name string, base string, unique bool) string {
	if !ensureItemArt() {
		return ""
	}
	if unique {
		if layout := poe.UniqueStashLayoutByName(name); layout != nil {
			if v := layout.GetItemVisualIdentity(); v != nil {
				return v.ArtPath()
			}
		}
	}
	return artPath(baseTypeByName[base])
}

// GemArtPath is the game art path of the gem item named name (a base, Vaal or transfigured gem).
func GemArtPath(name string) string {
	if !ensureItemArt() {
		return ""
	}
	gameID := gemGameIDByArt[strings.ToLower(name)]
	if gameID == "" {
		return ""
	}
	if b := poe.BaseItemTypeByIDMap[gameID]; b != nil {
		return artPath(b)
	}
	return artPath(poe.BaseItemTypeByIDMap[strings.Replace(gameID, "/SkillGemSupport", "/SupportGem", 1)])
}
