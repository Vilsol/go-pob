package raw

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func repoRootFromHere(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	testza.AssertNoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("walked to the filesystem root without finding go.mod, so any sweep below " +
				"would scan nothing and pass vacuously")
		}

		dir = parent
	}
}

// TestDataModuleVersionMatchesGoMod is the whole reason DataModuleVersion can be a constant.
func TestDataModuleVersionMatchesGoMod(t *testing.T) {
	root := repoRootFromHere(t)

	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	testza.AssertNoError(t, err)

	re := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(DataModulePath) + `\s+(\S+)\s*$`)
	m := re.FindSubmatch(raw)
	testza.AssertNotNil(t, m, "go.mod carries no require line for "+DataModulePath+
		", so this guard is scanning the wrong file or the module was renamed")

	testza.AssertEqual(t, string(m[1]), DataModuleVersion,
		"DataModuleVersion is "+DataModuleVersion+" but go.mod requires "+string(m[1])+
			" — bump the constant in the SAME commit as the dependency, or a warm asset cache will "+
			"serve the previous release's artifacts for up to cacheExpiry (30 days)")
}

// TestEveryCacheCallSiteIsNamespaced enumerates the cache call sites FROM THE SOURCE.
func TestEveryCacheCallSiteIsNamespaced(t *testing.T) {
	root := repoRootFromHere(t)

	call := regexp.MustCompile(`\.(GetCache|SetCache|ExistsInCache)\(([^,)]*)`)
	lineComment := regexp.MustCompile(`(?m)//.*$`)

	var files []string

	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			base := info.Name()
			if base == "node_modules" || base == ".git" || base == "testdata" {
				return filepath.SkipDir
			}

			if path == filepath.Join(root, "storage") {
				return filepath.SkipDir
			}

			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}

		return nil
	})
	testza.AssertNoError(t, walkErr)

	var checked int

	for _, path := range files {
		src, err := os.ReadFile(path) //nolint:gosec // a path this walk produced, inside the repo
		testza.AssertNoError(t, err, "failed reading "+path)

		for _, m := range call.FindAllStringSubmatch(lineComment.ReplaceAllString(string(src), ""), -1) {
			checked++

			testza.AssertTrue(t, strings.Contains(m[2], "CacheKey"),
				path+" calls "+m[1]+"("+strings.TrimSpace(m[2])+") with a key that does not come "+
					"from raw.CacheKey — a republished asset would be a stale HIT there. Every "+
					"cache reader and writer outside package storage must namespace, or none can.")
		}
	}

	testza.AssertGreater(t, checked, 3,
		"expected at least the four known call sites (3 in data/raw/main.go, 3 in "+
			"data/tree_versions.go); finding fewer means the sweep is not reaching the source")
}

// TestCacheKeyChangesWithTheDataModuleVersion pins the behaviour rather than the spelling: the same URL under two releases must not collide.
func TestCacheKeyChangesWithTheDataModuleVersion(t *testing.T) {
	const url = "https://data.pob.gg/data/3.29/raw/AlternateTreeVersions.msgpack.br"

	testza.AssertNotEqual(t, CacheKey(url), url,
		"CacheKey returned the URL unchanged, so a republish under the same URL is still a HIT")
	testza.AssertTrue(t, strings.Contains(CacheKey(url), DataModuleVersion),
		"the key must carry the release that published the asset")
	testza.AssertTrue(t, strings.HasSuffix(CacheKey(url), url),
		"the URL must survive intact so a key is still traceable to its asset")
}
