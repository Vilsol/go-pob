package dat

import (
	"io/fs"

	"github.com/oriath-net/pogo/poefs/bundle"
	"github.com/pkg/errors"
)

var bundleLoader fs.FS

func GetBundleLoader() (fs.FS, error) {
	if bundleLoader == nil {
		var err error
		bundleLoader, err = bundle.NewLoader(NewWebFS())
		return bundleLoader, errors.Wrap(err, "failed to create a new loader")
	}

	return bundleLoader, nil
}
