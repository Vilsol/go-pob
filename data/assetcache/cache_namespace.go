package assetcache

const DataModuleVersion = "v0.5.18-0.20261117171323-a96d84da7b44"

const DataModulePath = "github.com/Vilsol/go-pob-data"

func CacheKey(url string) string {
	return DataModuleVersion + "|" + url
}
