package assetcache

const DataModuleVersion = "v0.5.15"

const DataModulePath = "github.com/Vilsol/go-pob-data"

func CacheKey(url string) string {
	return DataModuleVersion + "|" + url
}
