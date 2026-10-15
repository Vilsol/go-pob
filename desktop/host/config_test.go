package host

import (
	"path/filepath"
	"testing"
)

func TestConfigAndChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go-pob", "desktop.json")
	if c := LoadConfig(path); c.BuildsDir != "" {
		t.Fatalf("missing config: %+v", c)
	}
	if err := SaveConfig(path, Config{BuildsDir: "/chosen"}); err != nil {
		t.Fatal(err)
	}
	c := LoadConfig(path)
	pob := []PoBBuilds{{Dir: "/pob/Builds"}}
	for _, tc := range []struct {
		c    Config
		pob  []PoBBuilds
		want BuildsChoice
	}{
		{c, pob, BuildsChoice{Dir: "/chosen", Source: "setting"}},
		{Config{}, pob, BuildsChoice{Dir: "/pob/Builds", Source: "pob"}},
		{Config{}, nil, BuildsChoice{Dir: "/app", Source: "app"}},
	} {
		got := ChooseBuilds(tc.c, tc.pob, "/app")
		if got.Dir != tc.want.Dir || got.Source != tc.want.Source {
			t.Errorf("%+v: %+v, want %+v", tc.c, got, tc.want)
		}
	}
}
