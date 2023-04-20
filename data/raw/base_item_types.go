package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type BaseItemType struct {
	raw2.BaseItemType
}

var BaseItemTypes []*BaseItemType

var BaseItemTypeByIDMap map[string]*BaseItemType

func InitializeBaseItemTypes(version string) error {
	return InitHelper(version, "BaseItemTypes", &BaseItemTypes, func(count int64) {
		BaseItemTypeByIDMap = make(map[string]*BaseItemType, count)
	}, func(obj *BaseItemType) {
		BaseItemTypeByIDMap[obj.ID] = obj
	})
}

func (b *BaseItemType) SkillGem() *SkillGem {
	return skillGemsByBaseItemTypeMap[b.Key]
}
