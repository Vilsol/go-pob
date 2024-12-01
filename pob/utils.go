package pob

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"slices"
	"strings"
)

func CompressEncode(xml string) (string, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)

	writer := zlib.NewWriter(encoder)
	if _, err := writer.Write([]byte(xml)); err != nil {
		return "", fmt.Errorf("failed to write to base64 encoder: %w", err)
	}

	writer.Close()
	encoder.Close()

	code := buf.String()
	code = strings.ReplaceAll(code, "+", "-")
	code = strings.ReplaceAll(code, "/", "_")

	return code, nil
}

func DecodeDecompress(code string) (string, error) {
	code = strings.ReplaceAll(code, "-", "+")
	code = strings.ReplaceAll(code, "_", "/")

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(code))
	reader, err := zlib.NewReader(decoder)
	if err != nil {
		return "", fmt.Errorf("failed to create a zlib reader: %w", err)
	}

	defer reader.Close()

	xml, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read from zlib reader: %w", err)
	}

	return string(xml), nil
}

func removeValue[T comparable](s []T, v T) ([]T, error) {
	var idx = slices.Index(s, v)
	if idx == -1 {
		return nil, fmt.Errorf("value %v was not found in slice", v)
	}

	s[idx] = s[len(s)-1]
	return s[:len(s)-1], nil
}
