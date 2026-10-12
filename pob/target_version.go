package pob

//pob:block:src/Modules/Build.lua:99-106,src/Modules/Build.lua:959
func (b *PathOfBuilding) NeedsVersionConversion() bool {
	return b.Build.TargetVersion != LiveTargetVersion
}

//pob:block:src/Modules/Build.lua:1340-1344
func (b *PathOfBuilding) ConvertToLiveVersion() {
	b.Build.TargetVersion = LiveTargetVersion
}
