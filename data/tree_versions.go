package data

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Vilsol/go-pob/cache"
)

type TreeVersion string

const (
	TreeVersion3_10 = TreeVersion("3_10")
	TreeVersion3_11 = TreeVersion("3_11")
	TreeVersion3_12 = TreeVersion("3_12")
	TreeVersion3_13 = TreeVersion("3_13")
	TreeVersion3_14 = TreeVersion("3_14")
	TreeVersion3_15 = TreeVersion("3_15")
	TreeVersion3_16 = TreeVersion("3_16")
	TreeVersion3_17 = TreeVersion("3_17")
	TreeVersion3_18 = TreeVersion("3_18")
)

const LatestTreeVersion = TreeVersion3_18
const DefaultTreeVersion = TreeVersion3_10

type TreeVersionData struct {
	Display    string
	Num        float64
	URL        string
	cachedTree *Tree
	rawTree    []byte
}

const cdnTreeBase = "https://go-pob-data.pages.dev/data/%s/tree/data.json.br"

func (v *TreeVersionData) Tree() *Tree {
	if v.cachedTree != nil {
		return v.cachedTree
	}

	var outTree Tree
	if err := json.Unmarshal(v.RawTree(), &outTree); err != nil {
		panic(errors.Wrap(err, "failed to decode file"))
	}
	v.cachedTree = &outTree

	return v.cachedTree
}

func (v *TreeVersionData) RawTree() []byte {
	if v.rawTree != nil {
		return v.rawTree
	}

	treeURL := fmt.Sprintf(cdnTreeBase, v.Display)
	var compressedTree []byte
	if cache.Disk().Exists(treeURL) {
		var err error
		compressedTree, err = cache.Disk().Get(treeURL)
		if err != nil {
			panic(err)
		}
	} else {
		log.Debug().Str("url", treeURL).Msg("fetching")
		response, err := http.DefaultClient.Get(treeURL)
		if err != nil {
			panic(errors.Wrap(err, "failed to fetch url: "+treeURL))
		}
		defer response.Body.Close()

		compressedTree, err = io.ReadAll(response.Body)
		if err != nil {
			panic(errors.Wrap(err, "failed to read response body"))
		}

		defer func() {
			_ = cache.Disk().Set(treeURL, compressedTree)
		}()
	}

	unzipStream := brotli.NewReader(bytes.NewReader(compressedTree))

	var err error
	v.rawTree, err = io.ReadAll(unzipStream)
	if err != nil {
		panic(errors.Wrap(err, "failed to read unzipped data"))
	}

	return v.rawTree
}

var TreeVersions = make(map[TreeVersion]*TreeVersionData)
