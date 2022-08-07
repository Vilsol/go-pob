package pob

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
	"strings"

	"github.com/pkg/errors"
)

func CompressEncode(xml string) (string, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)

	writer := zlib.NewWriter(encoder)
	if _, err := writer.Write([]byte(xml)); err != nil {
		return "", errors.Wrap(err, "failed to write to base64 encoder")
	}

	writer.Close()
	encoder.Close()

	println(buf.Len())
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
		return "", errors.Wrap(err, "failed to create a zlib reader")
	}

	defer reader.Close()

	xml, err := io.ReadAll(reader)
	if err != nil {
		return "", errors.Wrap(err, "failed to read from zlib reader")
	}

	return string(xml), nil
}
