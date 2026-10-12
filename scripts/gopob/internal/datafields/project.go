package datafields

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/Vilsol/go-pob-data/fields"
	"github.com/Vilsol/go-pob-data/loader"
	"github.com/andybalholm/brotli"
)

func AssetURL(version, path string) (string, error) {
	t, ok := fields.RowType(path)
	if !ok {
		return "", fmt.Errorf("asset path %q is not one go-pob-data's loader reads", path)
	}
	return loader.VariantURL(context.Background(), version, path, reflect.New(t).Interface()), nil
}

func Decompress(b []byte) ([]byte, error) {
	out, err := io.ReadAll(brotli.NewReader(bytes.NewReader(b)))
	if err != nil {
		return nil, fmt.Errorf("brotli: %w", err)
	}
	return out, nil
}

func Compress(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := brotli.NewWriterOptions(&buf, brotli.WriterOptions{Quality: 11, LGWin: 24})
	if _, err := w.Write(b); err != nil {
		return nil, fmt.Errorf("brotli: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("brotli: %w", err)
	}
	return buf.Bytes(), nil
}
