package builds

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/davecgh/go-spew/spew"
)

func TestParseBuild(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := ParseBuild(file)
	testza.AssertNoError(t, err)

	spew.Dump(build.Skills)
}
