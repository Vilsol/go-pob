package pob

//pob:block:src/Modules/Build.lua:103-110,src/Modules/Build.lua:977
func (b *PathOfBuilding) NeedsVersionConversion() bool {
	return b.Build.TargetVersion != LiveTargetVersion
}

//pob:block:src/Modules/Build.lua:1409-1413
func (b *PathOfBuilding) ConvertToLiveVersion() {
	b.Build.TargetVersion = LiveTargetVersion
}
