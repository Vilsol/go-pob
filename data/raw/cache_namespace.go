package raw

const DataModuleVersion = "v0.5.6"

const DataModulePath = "github.com/Vilsol/go-pob-data"

func CacheKey(url string) string {
	return DataModuleVersion + "|" + url
}
