package dat

import (
	"io/fs"

	"github.com/oriath-net/pogo/poefs/bundle"
)

var bundleLoader fs.FS

func GetBundleLoader() (fs.FS, error) {
	if bundleLoader == nil {
		var err error
		bundleLoader, err = bundle.NewLoader(NewWebFS())
		return bundleLoader, err
	}

	return bundleLoader, nil
}
