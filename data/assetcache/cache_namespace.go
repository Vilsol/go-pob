package assetcache

const DataModuleVersion = "v0.5.18-0.20261116221402-5ea6c648b7d5"

const DataModulePath = "github.com/Vilsol/go-pob-data"

func CacheKey(url string) string {
	return DataModuleVersion + "|" + url
}
