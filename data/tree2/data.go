package tree2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/andybalholm/brotli"
)

const dataBase = "https://data.pob.gg/data/"

// DataURL is where go-pob-data publishes the passive tree of treeVersion ("0_5" is game version 0.5).
func DataURL(treeVersion string) string {
	return dataBase + strings.ReplaceAll(treeVersion, "_", ".") + "/tree/data.json.br"
}

// LoadData loads treeVersion's passive tree from go-pob-data, through cache. The tree carries no
// sprites or assets.
func LoadData(ctx context.Context, treeVersion string, cache loader.AssetCache) (*PassiveTree, error) {
	url := DataURL(treeVersion)
	compressed, err := fetch(ctx, url, cache)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(brotli.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return nil, fmt.Errorf("tree2: decompressing %s: %w", url, err)
	}
	return Load(treeVersion, raw)
}

func fetch(ctx context.Context, url string, cache loader.AssetCache) ([]byte, error) {
	if cache.Exists(url) {
		if b, err := cache.Get(url); err == nil && len(b) > 0 {
			return b, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("tree2: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tree2: fetching %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tree2: fetching %s: %s", url, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tree2: reading %s: %w", url, err)
	}
	_ = cache.Set(url, b)
	return b, nil
}
