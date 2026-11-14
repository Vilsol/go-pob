package mcp

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/internal/importproxy"
	"github.com/Vilsol/go-pob/pob"
)

const maxImportBytes = 8 << 20

// fetchSource turns an import source into build XML: a file path, raw XML, a build-site URL from the import proxy's
// site table, or a PoB code.
func fetchSource(src string) (string, string, *ToolError) {
	if filepath.IsAbs(src) && strings.HasSuffix(strings.ToLower(src), ".xml") {
		b, err := os.ReadFile(src)
		if err != nil {
			e := errf(CodeImportFailed, "Cannot read %s: %v", src, err)
			e.Hint = "Pass the build code or XML instead."
			return "", "", e
		}
		return string(b), "file", nil
	}
	if strings.HasPrefix(src, "<") {
		return src, "xml", nil
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		code, e := fetchURL(src)
		if e != nil {
			return "", "", e
		}
		xml, err := pob.DecodeDecompress(strings.TrimSpace(code))
		if err != nil {
			e := errf(CodeImportFailed, "The site returned something that is not a PoB code.")
			e.Hint = "Paste the build code instead."
			return "", "", e
		}
		return xml, "url", nil
	}
	xml, err := pob.DecodeDecompress(src)
	if err != nil {
		e := errf(CodeParseError, "The source is not a PoB code, URL, XML or .xml path: %v", err)
		e.Hint = "Pass the whole PoB export code."
		return "", "", e
	}
	return xml, "code", nil
}

func fetchURL(url string) (string, *ToolError) {
	for _, site := range importproxy.SITES {
		if site.Download == "" || site.Game == "poe2" {
			continue
		}
		if ok, _ := regexp.MatchString(site.Match, url); !ok {
			continue
		}
		m := regexp.MustCompile(site.Regex).FindStringSubmatch(url)
		if len(m) < 2 {
			continue
		}
		target := "https://" + strings.ReplaceAll(site.Download, "$1", m[1])
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return "", importFailed(site.Label, 0, err.Error())
		}
		req.Header.Set("User-Agent", "go-pob")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", importFailed(site.Label, 0, err.Error())
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return "", importFailed(site.Label, res.StatusCode, res.Status)
		}
		b, err := io.ReadAll(io.LimitReader(res.Body, maxImportBytes))
		if err != nil {
			return "", importFailed(site.Label, 0, err.Error())
		}
		return string(b), nil
	}
	var labels []string
	for _, s := range importproxy.SITES {
		if s.Download != "" && s.Game != "poe2" {
			labels = append(labels, s.Label)
		}
	}
	e := errf(CodeImportFailed, "No supported build site matches %s.", url)
	e.Valid = labels
	e.Hint = "Paste the build code instead."
	return "", e
}

func importFailed(site string, status int, msg string) *ToolError {
	e := errf(CodeImportFailed, "%s did not return the build: %s", site, msg)
	e.Site = site
	e.Status = status
	e.Hint = "Paste the build code instead."
	return e
}
