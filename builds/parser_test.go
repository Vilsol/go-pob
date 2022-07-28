package builds

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/davecgh/go-spew/spew"
)

func TestParseBuild(t *testing.T) {
	file, _ := os.ReadFile("./testdata/builds/Fireball.xml")
	build, err := ParseBuild(file)
	testza.AssertNoError(t, err)

	spew.Dump(build.Skills)
}
