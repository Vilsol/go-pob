package exposition

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"strings"
)

// ExportCode is the build as Save writes it, deflated and base64url-encoded.
//
//pob2:block:src/Classes/ImportTab.lua:135-137
func (c *Calculator) ExportCode(player []SavedStat, fullDPS []SavedFullDPS, minion []SavedStat) (string, error) {
	xml, err := c.SerializeWithStats(player, fullDPS, minion)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte(xml)); err != nil {
		return "", fmt.Errorf("deflating the build: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("deflating the build: %w", err)
	}
	code := base64.StdEncoding.EncodeToString(buf.Bytes())
	return strings.NewReplacer("+", "-", "/", "_").Replace(code), nil
}
