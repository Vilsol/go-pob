package calculator

import (
	"sync"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

var (
	globalItemMu    sync.RWMutex
	globalItemCache = make(map[string]*Item)
)

func itemTextByID(env *Environment) map[int]string {
	if env.itemText == nil {
		env.itemText = make(map[int]string, len(env.Build.Items.Items))
		for _, it := range env.Build.Items.Items {
			env.itemText[it.ID] = it.Text
		}
	}
	return env.itemText
}

func parseItemCached(cache *EnvironmentCache, raw string) *Item {
	return peekItemCached(cache, raw).Clone()
}

func peekItemCached(cache *EnvironmentCache, raw string) *Item {
	if cache != nil {
		cache.mu.RLock()
		master, ok := cache.items[raw]
		keeps := cache.items != nil
		cache.mu.RUnlock()
		if ok {
			return master
		}
		if keeps {
			master = parseItemGlobal(raw)
			cache.mu.Lock()
			if kept, ok := cache.items[raw]; ok {
				master = kept
			} else {
				cache.items[raw] = master
			}
			cache.mu.Unlock()

			return master
		}
	}

	return parseItemGlobal(raw)
}

func PeekItem(raw string) *Item {
	return parseItemGlobal(raw)
}

func parseItemGlobal(raw string) *Item {
	globalItemMu.RLock()
	master, ok := globalItemCache[raw]
	globalItemMu.RUnlock()
	if ok {
		return master
	}

	master = ParseItem(raw)

	globalItemMu.Lock()
	globalItemCache[raw] = master
	globalItemMu.Unlock()

	return master
}

func (i *Item) Clone() *Item {
	if i == nil {
		return nil
	}

	c := *i

	if i.slotModList2 != nil {
		seen := make(map[mod.Mod]mod.Mod, len(i.Mods)+len(i.slotModList2)+len(i.slotModList3))
		c.Mods = cloneModsShared(i.Mods, seen)
		c.slotModList2 = cloneModsShared(i.slotModList2, seen)
		if i.slotModList3 != nil {
			c.slotModList3 = cloneModsShared(i.slotModList3, seen)
		}
	} else {
		c.Mods = mod.CloneSlice(i.Mods)
	}
	c.BuffModList = mod.CloneSlice(i.BuffModList)

	c.Requirements = cloneFloatMap(i.Requirements)
	c.ArmourData = cloneFloatMap(i.ArmourData)
	c.FlaskData = cloneFloatMap(i.FlaskData)

	if i.WeaponData != nil {
		c.WeaponData = make(map[int]map[string]float64, len(i.WeaponData))
		for k, v := range i.WeaponData {
			c.WeaponData[k] = cloneFloatMap(v)
		}
	}

	if i.Sockets != nil {
		c.Sockets = make([]ItemSocket, len(i.Sockets))
		copy(c.Sockets, i.Sockets)
	}
	if i.VariantList != nil {
		c.VariantList = make([]string, len(i.VariantList))
		copy(c.VariantList, i.VariantList)
	}

	c.Quality = clonePtr(i.Quality)
	c.ItemLevel = clonePtr(i.ItemLevel)
	c.Variant = clonePtr(i.Variant)
	c.VariantAlt = clonePtr(i.VariantAlt)
	c.VariantAlt2 = clonePtr(i.VariantAlt2)
	c.VariantAlt3 = clonePtr(i.VariantAlt3)
	c.VariantAlt4 = clonePtr(i.VariantAlt4)
	c.VariantAlt5 = clonePtr(i.VariantAlt5)
	c.Catalyst = clonePtr(i.Catalyst)
	c.CatalystQuality = clonePtr(i.CatalystQuality)
	c.TalismanTier = clonePtr(i.TalismanTier)
	c.Limit = clonePtr(i.Limit)

	return &c
}

func cloneModsShared(mods []mod.Mod, seen map[mod.Mod]mod.Mod) []mod.Mod {
	if mods == nil {
		return nil
	}

	out := make([]mod.Mod, len(mods))
	for idx, m := range mods {
		clone, ok := seen[m]
		if !ok {
			clone = m.Clone()
			seen[m] = clone
		}
		out[idx] = clone
	}

	return out
}

func cloneFloatMap[K comparable](m map[K]float64) map[K]float64 {
	if m == nil {
		return nil
	}

	out := make(map[K]float64, len(m))
	for k, v := range m {
		out[k] = v
	}

	return out
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}

	v := *p
	return &v
}

func (c *EnvironmentCache) useTreeVersion(v pob.TreeVersion) {
	c.mu.RLock()
	same := c.TreeVersion == v && c.modsForNodes != nil
	c.mu.RUnlock()
	if same {
		return
	}

	c.mu.Lock()
	if c.TreeVersion != v || c.modsForNodes == nil {
		c.modsForNodes = make(map[string]moddb.ModList, len(data.TreeVersions[v].Tree().Nodes))
		c.TreeVersion = v
	}
	c.mu.Unlock()
}

func (c *EnvironmentCache) nodeMods(nodeID string) (moddb.ModList, bool) {
	c.mu.RLock()
	list, ok := c.modsForNodes[nodeID]
	c.mu.RUnlock()

	return list, ok
}

func (c *EnvironmentCache) storeNodeMods(nodeID string, list moddb.ModList) {
	c.mu.Lock()
	if _, ok := c.modsForNodes[nodeID]; !ok {
		c.modsForNodes[nodeID] = list
	}
	c.mu.Unlock()
}
