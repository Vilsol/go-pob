package exposition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/storage"
)

type treeArtOption struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

type treeArtNodeIn struct {
	Icon              string          `json:"icon"`
	ActiveEffectImage string          `json:"activeEffectImage"`
	NodeOverlay       json.RawMessage `json:"nodeOverlay"`
	ConnectionArt     string          `json:"connectionArt"`
	Options           json.RawMessage `json:"options"`
}

type treeArtNode struct {
	Icon              string          `json:"icon,omitempty"`
	ActiveEffectImage string          `json:"activeEffectImage,omitempty"`
	NodeOverlay       json.RawMessage `json:"nodeOverlay,omitempty"`
	ConnectionArt     string          `json:"connectionArt,omitempty"`
	// Options are an attribute node's choices, or a switchable node's per-class replacements, by name and icon.
	Options []treeArtOption `json:"options,omitempty"`
}

type treeArt struct {
	Sprites       json.RawMessage        `json:"sprites,omitempty"`
	NodeOverlay   json.RawMessage        `json:"nodeOverlay,omitempty"`
	ConnectionArt json.RawMessage        `json:"connectionArt,omitempty"`
	Classes       json.RawMessage        `json:"classes,omitempty"`
	Nodes         map[string]treeArtNode `json:"nodes"`
}

var (
	treeArtMu  sync.Mutex
	treeArtDoc json.RawMessage
)

// treeArtJSON is the art the view draws by name from the tree's data.json (PassiveTree.lua:144-189, 547-568,
// 630-633): the sprite sheets, the frame and line names, the class and ascendancy backgrounds, and each node's
// icon, effect, frame, connection art and option icons. It reads the copy the tree loaded through the cache.
func treeArtJSON() (json.RawMessage, error) {
	treeArtMu.Lock()
	defer treeArtMu.Unlock()
	if treeArtDoc != nil {
		return treeArtDoc, nil
	}
	url := tree2.DataURL(TreeVersion)
	cache := assetcache.AssetLoaderWrapper{Storage: storage.Get()}
	var compressed []byte
	if cache.Exists(url) {
		compressed, _ = cache.Get(url)
	}
	if len(compressed) == 0 {
		resp, err := http.Get(url) //nolint:gosec,noctx
		if err != nil {
			return nil, fmt.Errorf("tree art: fetching %s: %w", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("tree art: fetching %s: %s", url, resp.Status)
		}
		if compressed, err = io.ReadAll(resp.Body); err != nil {
			return nil, fmt.Errorf("tree art: reading %s: %w", url, err)
		}
	}
	raw, err := io.ReadAll(brotli.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return nil, fmt.Errorf("tree art: decompressing %s: %w", url, err)
	}
	var in struct {
		Sprites       json.RawMessage          `json:"sprites"`
		NodeOverlay   json.RawMessage          `json:"nodeOverlay"`
		ConnectionArt json.RawMessage          `json:"connectionArt"`
		Classes       json.RawMessage          `json:"classes"`
		Nodes         map[string]treeArtNodeIn `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("tree art: decoding %s: %w", url, err)
	}
	out := treeArt{Sprites: in.Sprites, NodeOverlay: in.NodeOverlay, ConnectionArt: in.ConnectionArt, Classes: in.Classes, Nodes: map[string]treeArtNode{}}
	for id, n := range in.Nodes {
		node := treeArtNode{Icon: n.Icon, ActiveEffectImage: n.ActiveEffectImage, NodeOverlay: n.NodeOverlay, ConnectionArt: n.ConnectionArt}
		// an attribute node lists its options, a switchable node keys them by class
		var list []treeArtOption
		if json.Unmarshal(n.Options, &list) != nil {
			var byClass map[string]treeArtOption
			if json.Unmarshal(n.Options, &byClass) == nil {
				for _, o := range byClass {
					list = append(list, o)
				}
				slices.SortFunc(list, func(a, b treeArtOption) int { return strings.Compare(a.Name, b.Name) })
			}
		}
		node.Options = list
		out.Nodes[id] = node
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("tree art: encoding: %w", err)
	}
	treeArtDoc = b
	return b, nil
}
