package assetcache

const DataModuleVersion = "v0.5.18-0.20261118190629-100f534cb9f8"

const DataModulePath = "github.com/Vilsol/go-pob-data"

func CacheKey(url string) string {
	return DataModuleVersion + "|" + url
}
