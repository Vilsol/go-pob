package assetcache

const DataModuleVersion = "v0.5.18-0.20261113220728-84c27d6ae28d"

const DataModulePath = "github.com/Vilsol/go-pob-data"

func CacheKey(url string) string {
	return DataModuleVersion + "|" + url
}
