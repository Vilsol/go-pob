package raw

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/andybalholm/brotli"

	"github.com/Vilsol/go-pob/cache"
)

const cdnBase = "https://go-pob-data.pages.dev/data/%s/%s.json.br"

// LoadRaw loads a raw brotli-compressed json dump from remote source
//
// Returns data from cache if found
func LoadRaw[T any](version string, name string) (*T, error) {
	url := fmt.Sprintf(cdnBase, version, name)

	var b []byte
	if cache.Disk().Exists(url) {
		get, err := cache.Disk().Get(url)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve url from cache: "+url)
		}
		b = get
	} else {
		log.Debug().Str("url", url).Msg("fetching")
		response, err := http.DefaultClient.Get(url)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch url: "+url)
		}
		defer response.Body.Close()

		b, err = io.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read response body")
		}

		defer func() {
			_ = cache.Disk().Set(url, b)
		}()
	}

	unzipStream := brotli.NewReader(bytes.NewReader(b))
	unzipped, err := io.ReadAll(unzipStream)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read unzipped data")
	}

	out := new(T)
	if err := json.Unmarshal(unzipped, out); err != nil {
		return nil, errors.Wrap(err, "failed to decode file")
	}

	return out, nil
}

func InitHelper[T any](version string, name string, target *T) error {
	loadedRaw, err := LoadRaw[T](version, name)
	if err != nil {
		return err
	}

	*target = *loadedRaw

	return nil
}
