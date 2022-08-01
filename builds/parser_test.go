package builds

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestParseBuild(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	_, err = ParseBuild(file)
	testza.AssertNoError(t, err)
}
