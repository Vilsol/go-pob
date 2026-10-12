package raw

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob-data/loader"
	rawdata "github.com/Vilsol/go-pob-data/raw"
	tjdata "github.com/Vilsol/timeless-jewels/data"
)

var (
	AlternateTreeVersions     []*rawdata.AlternateTreeVersion
	PassiveSkills             []*rawdata.PassiveSkill
	AlternatePassiveSkills    []*rawdata.AlternatePassiveSkill
	AlternatePassiveAdditions []*rawdata.AlternatePassiveAddition
)

var timelessTables = []string{"AlternateTreeVersions", "PassiveSkills", "AlternatePassiveSkills", "AlternatePassiveAdditions"}

var passiveSkillByGraphID map[int64]*rawdata.PassiveSkill

func AlternateTreeVersionByConqueror(conquerorType string) *rawdata.AlternateTreeVersion {
	for _, v := range AlternateTreeVersions {
		if strings.EqualFold(v.ConquerorType, conquerorType) {
			return v
		}
	}
	return nil
}

func JewelTypeForConqueror(conquerorType string) int {
	if v := AlternateTreeVersionByConqueror(conquerorType); v != nil {
		return v.Key
	}
	return 0
}

func PassiveSkillByGraphID(graphID int64) *rawdata.PassiveSkill {
	return passiveSkillByGraphID[graphID]
}

var (
	timelessOnce    sync.Once
	errTimelessLoad error
)

func EnsureTimelessSources() error {
	timelessOnce.Do(func() {
		if len(AlternateTreeVersions) > 0 {
			return
		}
		errTimelessLoad = initTimelessSources(DataContext(context.Background()), LatestVersion, lazyLoader())
	})
	return errTimelessLoad
}

func TimelessSourcesLoaded() bool {
	return len(AlternateTreeVersions) > 0 && len(PassiveSkills) > 0
}

func initTimelessSources(ctx context.Context, version string, cache loader.AssetCache) error {
	if err := loader.InitHelper(ctx, version, "AlternateTreeVersions", &AlternateTreeVersions, nil, cache); err != nil {
		return err //nolint:wrapcheck
	}

	passiveSkillByGraphID = make(map[int64]*rawdata.PassiveSkill)
	if err := loader.InitHelper(ctx, version, "PassiveSkills", &PassiveSkills, nil, cache,
		func(ps *rawdata.PassiveSkill) { passiveSkillByGraphID[ps.Hash] = ps }); err != nil {
		return err //nolint:wrapcheck
	}

	if err := loader.InitHelper(ctx, version, "AlternatePassiveSkills", &AlternatePassiveSkills, nil, cache); err != nil {
		return err //nolint:wrapcheck
	}

	if err := loader.InitHelper(ctx, version, "AlternatePassiveAdditions", &AlternatePassiveAdditions, nil, cache); err != nil {
		return err //nolint:wrapcheck
	}

	return initTimelessJewelCalculator()
}

type conv struct {
	err error
}

func (c *conv) fail(field string, v int64) {
	if c.err == nil {
		c.err = fmt.Errorf("timeless: %s = %d is out of range for the library's column width", field, v)
	}
}

func (c *conv) u32(field string, v int64) uint32 {
	if v < 0 || v > math.MaxUint32 {
		c.fail(field, v)
		return 0
	}
	return uint32(v)
}

func (c *conv) i32(field string, v int64) int32 {
	if v < math.MinInt32 || v > math.MaxInt32 {
		c.fail(field, v)
		return 0
	}
	return int32(v)
}

func initTimelessJewelCalculator() error {
	if len(PassiveSkills) == 0 {
		return fmt.Errorf("timeless: PassiveSkills is empty; initTimelessSources must run first")
	}

	c := &conv{}

	passives := make([]*tjdata.PassiveSkill, 0, len(PassiveSkills))
	for _, p := range PassiveSkills {
		stats := make([]uint32, 0, len(p.Stats))
		for _, s := range p.Stats {
			stats = append(stats, c.u32("PassiveSkills.Stats", s))
		}

		passives = append(passives, &tjdata.PassiveSkill{
			Index:               c.u32("PassiveSkills._key", p.Key),
			ID:                  p.ID,
			StatIndices:         stats,
			PassiveSkillGraphID: c.u32("PassiveSkills.PassiveSkillGraphId", p.Hash),
			Name:                p.Name,
			IsKeystone:          p.Keystone,
			IsNotable:           p.Notable,
			IsJewelSocket:       p.JewelSocket,
			AscendancyKey:       optionalU32(c, "PassiveSkills.AscendancyKey", p.Ascendancy),
			DescendancyKey:      optionalU32(c, "PassiveSkills.DescendancyKey", p.Descendancy),
		})
	}

	altSkills := make([]*tjdata.AlternatePassiveSkill, 0, len(AlternatePassiveSkills))
	for _, a := range AlternatePassiveSkills {
		altSkills = append(altSkills, &tjdata.AlternatePassiveSkill{
			Index:                    c.u32("AlternatePassiveSkills._key", int64(a.Key)),
			ID:                       a.ID,
			AlternateTreeVersionsKey: c.u32("AlternatePassiveSkills.AlternateTreeVersionsKey", int64(a.AlternateTreeVersionsKey)),
			Name:                     a.Name,
			PassiveType:              passiveTypes(c, "AlternatePassiveSkills.PassiveType", a.PassiveType),
			StatsKeys:                statsKeys(c, "AlternatePassiveSkills.StatsKeys", a.StatsKeys),
			Stat1Min:                 c.i32("AlternatePassiveSkills.Stat1Min", int64(a.Stat1Min)),
			Stat1Max:                 c.i32("AlternatePassiveSkills.Stat1Max", int64(a.Stat1Max)),
			Stat2Min:                 c.i32("AlternatePassiveSkills.Stat2Min", int64(a.Stat2Min)),
			Stat2Max:                 c.i32("AlternatePassiveSkills.Stat2Max", int64(a.Stat2Max)),
			Stat3Min:                 c.i32("AlternatePassiveSkills.Var9", int64(a.Stat3Min)),
			Stat3Max:                 c.i32("AlternatePassiveSkills.Var10", int64(a.Stat3Max)),
			Stat4Min:                 c.i32("AlternatePassiveSkills.Var11", int64(a.Stat4Min)),
			Stat4Max:                 c.i32("AlternatePassiveSkills.Var12", int64(a.Stat4Max)),
			SpawnWeight:              c.u32("AlternatePassiveSkills.SpawnWeight", int64(a.SpawnWeight)),
			ConquerorIndex:           c.u32("AlternatePassiveSkills.Var18", int64(a.ConquerorIndex)),
			RandomMin:                c.u32("AlternatePassiveSkills.RandomMin", int64(a.RandomMin)),
			RandomMax:                c.u32("AlternatePassiveSkills.RandomMax", int64(a.RandomMax)),
			ConquerorVersion:         c.u32("AlternatePassiveSkills.Var24", int64(a.ConquerorVersion)),
		})
	}

	altAdditions := make([]*tjdata.AlternatePassiveAddition, 0, len(AlternatePassiveAdditions))
	for _, a := range AlternatePassiveAdditions {
		altAdditions = append(altAdditions, &tjdata.AlternatePassiveAddition{
			Index:                    c.u32("AlternatePassiveAdditions._key", int64(a.Key)),
			ID:                       a.ID,
			AlternateTreeVersionsKey: c.u32("AlternatePassiveAdditions.AlternateTreeVersionsKey", int64(a.AlternateTreeVersionsKey)),
			SpawnWeight:              c.u32("AlternatePassiveAdditions.SpawnWeight", int64(a.SpawnWeight)),
			StatsKeys:                statsKeys(c, "AlternatePassiveAdditions.StatsKeys", a.StatsKeys),
			Stat1Min:                 c.i32("AlternatePassiveAdditions.Stat1Min", int64(a.Stat1Min)),
			Stat1Max:                 c.i32("AlternatePassiveAdditions.Stat1Max", int64(a.Stat1Max)),
			Stat2Min:                 c.i32("AlternatePassiveAdditions.Var6", int64(a.Stat2Min)),
			Stat2Max:                 c.i32("AlternatePassiveAdditions.Var7", int64(a.Stat2Max)),
			PassiveType:              passiveTypes(c, "AlternatePassiveAdditions.PassiveType", a.PassiveType),
		})
	}

	treeVersions := make([]*tjdata.AlternateTreeVersion, 0, len(AlternateTreeVersions))
	for _, v := range AlternateTreeVersions {
		treeVersions = append(treeVersions, &tjdata.AlternateTreeVersion{
			Index:                                  c.u32("AlternateTreeVersions._key", int64(v.Key)),
			ID:                                     v.ConquerorType,
			AreSmallAttributePassiveSkillsReplaced: v.AreSmallAttributePassiveSkillsReplaced,
			AreSmallNormalPassiveSkillsReplaced:    v.AreSmallNormalPassiveSkillsReplaced,
			MinimumAdditions:                       c.u32("AlternateTreeVersions.Var5", int64(v.MinimumAdditions)),
			MaximumAdditions:                       c.u32("AlternateTreeVersions.Var6", int64(v.MaximumAdditions)),
			MinimumAdditionsSmallAttribute:         c.u32("AlternateTreeVersions.Var3", int64(v.MinimumAdditionsSmallAttribute)),
			MaximumAdditionsSmallAttribute:         c.u32("AlternateTreeVersions.Var4", int64(v.MaximumAdditionsSmallAttribute)),
			MinimumAdditionsNotable:                c.u32("AlternateTreeVersions.Var7", int64(v.MinimumAdditionsNotable)),
			MaximumAdditionsNotable:                c.u32("AlternateTreeVersions.Var8", int64(v.MaximumAdditionsNotable)),
			NotableReplacementSpawnWeight:          c.u32("AlternateTreeVersions.Var9", int64(v.NotableReplacementSpawnWeight)),
		})
	}

	if c.err != nil {
		return c.err
	}

	if err := tjdata.Initialize(passives, altSkills, altAdditions, treeVersions); err != nil {
		return fmt.Errorf("timeless: %w", err)
	}

	return nil
}

func optionalU32(c *conv, field string, v *int64) *uint32 {
	if v == nil {
		return nil
	}
	u := c.u32(field, *v)
	return &u
}

func passiveTypes(c *conv, field string, in []int) []tjdata.PassiveSkillType {
	out := make([]tjdata.PassiveSkillType, 0, len(in))
	for _, t := range in {
		out = append(out, tjdata.PassiveSkillType(c.u32(field, int64(t))))
	}
	return out
}

func statsKeys(c *conv, field string, in []int) []uint32 {
	out := make([]uint32, 0, len(in))
	for _, k := range in {
		out = append(out, c.u32(field, int64(k)))
	}
	return out
}
