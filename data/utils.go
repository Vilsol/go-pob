package data

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
)

func FromJSONGz[T any](data []byte) T {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}

	all, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	var out = new(T)
	if err := json.Unmarshal(all, &out); err != nil {
		panic(err)
	}
	return *out
}
