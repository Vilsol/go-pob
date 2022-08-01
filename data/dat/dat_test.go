package dat

import (
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestReadIndex(t *testing.T) {
	_, err := GetBundleLoader()
	testza.AssertNoError(t, err)
}

func TestReadSkillGems(t *testing.T) {
	loader, err := GetBundleLoader()
	testza.AssertNoError(t, err)

	skillGems, err := loader.Open("Data/SkillGems.dat64")
	testza.AssertNoError(t, err)

	stat, err := skillGems.Stat()
	testza.AssertNoError(t, err)

	gemBytes := make([]byte, stat.Size())
	read, err := skillGems.Read(gemBytes)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, stat.Size(), int64(read))
}

func TestLoadSchema(t *testing.T) {
	LoadSchema()

	testza.AssertNotNil(t, schemaFile)
	testza.AssertEqual(t, int64(3), schemaFile.Version)

	schema := GetSchema("SkillGems")
	testza.AssertEqual(t, "SkillGems", schema.Name)
	testza.AssertEqual(t, 17, len(schema.Columns))
}

func TestParseDat(t *testing.T) {
	LoadParser()

	loader, err := GetBundleLoader()
	testza.AssertNoError(t, err)

	testFiles := map[string]int{
		"Data/SkillGems.dat64":      703,
		"Data/ActiveSkills.dat64":   1367,
		"Data/GrantedEffects.dat64": 10020,
	}

	for file, count := range testFiles {
		t.Run(file, func(t *testing.T) {
			data, err := loader.Open(file)
			testza.AssertNoError(t, err)

			dat, err := ParseDat(data, filepath.Base(file))
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, count, len(dat))
		})
	}
}
