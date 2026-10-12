package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

type node struct {
	Name  string   `json:"name"`
	Stats []string `json:"stats"`
}

type dump struct {
	Version string          `json:"version"`
	Nodes   map[string]node `json:"nodes"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tree_ours_dump <out.json> [tree version]")
		os.Exit(2)
	}

	config.InitLogging(false)
	if err := poe.InitializeAll(raw.DataContext(context.Background()), raw.LatestVersion,
		raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: game data %s failed to load: %v\n", raw.LatestVersion, err)
		os.Exit(3)
	}

	tv := data.LatestTreeVersion
	if len(os.Args) > 2 && os.Args[2] != "" {
		tv = pob.TreeVersion(os.Args[2])
	}
	vd, ok := data.TreeVersions[tv]
	if !ok {
		fmt.Fprintf(os.Stderr, "REFUSING: no tree registered for %q\n", tv)
		os.Exit(3)
	}
	tree := vd.Tree()
	if len(tree.Nodes) == 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: tree %s loaded with no nodes\n", tv)
		os.Exit(3)
	}

	out := dump{Version: string(tv), Nodes: map[string]node{}}
	for id, n := range tree.Nodes {
		name := ""
		if n.Name != nil {
			name = *n.Name
		}
		out.Nodes[id] = node{Name: name, Stats: n.Stats}
	}

	blob, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
		os.Exit(3)
	}
	//nolint:gosec // G703: os.Args[1] is the output path this operator tool was asked to write.
	if err := os.WriteFile(os.Args[1], blob, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
		os.Exit(3)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (tree %s, %d nodes)\n", os.Args[1], tv, len(out.Nodes))
}
