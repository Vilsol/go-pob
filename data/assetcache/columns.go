package assetcache

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
)

// Columns serves the json tables in Rows with only the columns their row types decode, and without the zero values
// decoding would leave as they are, so a load decodes a fraction of the bytes. The pruned copy is cached beside the
// original, under a key that fingerprints the row types and the pruning, and is rewritten whenever the original is.
type Columns struct {
	loader.AssetCache
	// Rows are the row types each table url is decoded into.
	Rows map[string][]reflect.Type
}

func (c Columns) prunedKey(url string, types []reflect.Type) string {
	return url + "#columns-" + columnsFingerprint(types)
}

func (c Columns) Get(url string) ([]byte, error) {
	types, ok := c.Rows[url]
	if !ok {
		b, err := c.AssetCache.Get(url)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", url, err)
		}
		return b, nil
	}
	key := c.prunedKey(url, types)
	if c.AssetCache.Exists(key) {
		if b, err := c.AssetCache.Get(key); err == nil && len(b) > 0 {
			return b, nil
		}
	}
	b, err := c.AssetCache.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	if len(b) == 0 {
		return b, nil
	}
	pruned, err := pruneTable(b, tableColumns(types))
	if err != nil {
		return b, nil //nolint:nilerr // an original that does not prune is decoded as it is
	}
	_ = c.AssetCache.Set(key, pruned)
	return pruned, nil
}

func (c Columns) Set(url string, b []byte) error {
	if err := c.AssetCache.Set(url, b); err != nil {
		return fmt.Errorf("%s: %w", url, err)
	}
	if types, ok := c.Rows[url]; ok {
		if pruned, err := pruneTable(b, tableColumns(types)); err == nil {
			_ = c.AssetCache.Set(c.prunedKey(url, types), pruned)
		}
	}
	return nil
}

// column is one json key a row type decodes, by its lower-case name; zero is the literal that decodes to the field's
// zero value, empty when none is dropped.
type column struct {
	zero string
}

func tableColumns(types []reflect.Type) map[string]column {
	out := map[string]column{}
	for _, t := range types {
		addColumns(out, t)
	}
	return out
}

// addColumns follows encoding/json's field rules: a tag name or the field name, "-" skipped, an untagged embedded
// struct's fields promoted.
func addColumns(out map[string]column, t reflect.Type) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	for i := range t.NumField() {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		name, opts, _ := strings.Cut(tag, ",")
		if name == "-" && opts == "" {
			continue
		}
		ft := f.Type
		if f.Anonymous && name == "" {
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				addColumns(out, ft)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}
		key := strings.ToLower(name)
		zero := ""
		if opts == "" {
			zero = zeroLiteral(ft.Kind())
		}
		if prev, seen := out[key]; seen && prev.zero != zero {
			zero = ""
		}
		out[key] = column{zero: zero}
	}
}

func zeroLiteral(k reflect.Kind) string {
	switch k {
	case reflect.Bool:
		return "false"
	case reflect.String:
		return `""`
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "0"
	}
	return ""
}

// columnsProbe is pruned into the fingerprint, so a change to what pruning keeps changes every pruned key.
const columnsProbe = `[{"_key":0,"Id":"a","B":false,"N":0,"S":"","P":null,"L":[],"O":{"x":[1,"]"]},"Drop":1,"F":0.5,"E":"\"}"}]`

type columnsProbeRow struct {
	Key int            `json:"_key"`
	ID  string         `json:"Id"`
	B   bool           `json:"B"`
	N   int            `json:"N"`
	S   string         `json:"S"`
	P   *int           `json:"P"`
	L   []int          `json:"L"`
	O   map[string]any `json:"O"`
	F   float64        `json:"F"`
	E   string         `json:"E"`
}

func columnsFingerprint(types []reflect.Type) string {
	h := sha256.New()
	probe, err := pruneJSON([]byte(columnsProbe), tableColumns([]reflect.Type{reflect.TypeFor[columnsProbeRow]()}))
	if err != nil {
		panic(err)
	}
	h.Write(probe)
	for _, t := range types {
		cols := tableColumns([]reflect.Type{t})
		fmt.Fprintf(h, "|%s", t.String())
		for _, f := range reflect.VisibleFields(t) {
			fmt.Fprintf(h, ";%s %s %s", f.Name, f.Type.String(), f.Tag.Get("json"))
		}
		fmt.Fprintf(h, ";%d", len(cols))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func pruneTable(gz []byte, cols map[string]column) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("decompressing: %w", err)
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decompressing: %w", err)
	}
	pruned, err := pruneJSON(raw, cols)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	w, _ := gzip.NewWriterLevel(&out, gzip.BestSpeed)
	if _, err := w.Write(pruned); err != nil {
		return nil, fmt.Errorf("compressing: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("compressing: %w", err)
	}
	return out.Bytes(), nil
}

var errNotTable = fmt.Errorf("not a json array of objects")

// pruneJSON keeps the keys of each row that cols names, other than a droppable zero or a null; a key with an escape
// in it is kept.
func pruneJSON(raw []byte, cols map[string]column) ([]byte, error) {
	s := jsonScan{b: raw}
	out := make([]byte, 0, len(raw)/4)
	var lower []byte
	s.space()
	if !s.take('[') {
		return nil, errNotTable
	}
	out = append(out, '[')
	s.space()
	if s.take(']') {
		return append(out, ']'), s.end()
	}
	for {
		s.space()
		if !s.take('{') {
			return nil, errNotTable
		}
		out = append(out, '{')
		kept := 0
		s.space()
		if !s.take('}') {
			for {
				s.space()
				keyStart := s.i
				escaped, err := s.str()
				if err != nil {
					return nil, err
				}
				key := s.b[keyStart:s.i]
				s.space()
				if !s.take(':') {
					return nil, errNotTable
				}
				s.space()
				valueStart := s.i
				if err := s.value(); err != nil {
					return nil, err
				}
				value := s.b[valueStart:s.i]
				keep := escaped
				if !escaped {
					lower = append(lower[:0], key[1:len(key)-1]...)
					for i, c := range lower {
						if 'A' <= c && c <= 'Z' {
							lower[i] = c + 'a' - 'A'
						}
					}
					col, ok := cols[string(lower)]
					keep = ok && string(value) != "null" && (col.zero == "" || string(value) != col.zero)
				}
				if keep {
					if kept > 0 {
						out = append(out, ',')
					}
					out = append(out, key...)
					out = append(out, ':')
					out = append(out, value...)
					kept++
				}
				s.space()
				if s.take(',') {
					continue
				}
				if !s.take('}') {
					return nil, errNotTable
				}
				break
			}
		}
		out = append(out, '}')
		s.space()
		if s.take(',') {
			out = append(out, ',')
			continue
		}
		if !s.take(']') {
			return nil, errNotTable
		}
		return append(out, ']'), s.end()
	}
}

type jsonScan struct {
	b []byte
	i int
}

func (s *jsonScan) space() {
	for s.i < len(s.b) && (s.b[s.i] == ' ' || s.b[s.i] == '\t' || s.b[s.i] == '\n' || s.b[s.i] == '\r') {
		s.i++
	}
}

func (s *jsonScan) take(c byte) bool {
	if s.i < len(s.b) && s.b[s.i] == c {
		s.i++
		return true
	}
	return false
}

func (s *jsonScan) end() error {
	s.space()
	if s.i != len(s.b) {
		return errNotTable
	}
	return nil
}

// str moves past a string and reports whether it held an escape.
func (s *jsonScan) str() (bool, error) {
	if !s.take('"') {
		return false, errNotTable
	}
	escaped := false
	for s.i < len(s.b) {
		switch s.b[s.i] {
		case '\\':
			escaped = true
			s.i += 2
		case '"':
			s.i++
			return escaped, nil
		default:
			s.i++
		}
	}
	return false, errNotTable
}

func (s *jsonScan) value() error {
	if s.i >= len(s.b) {
		return errNotTable
	}
	switch s.b[s.i] {
	case '"':
		_, err := s.str()
		return err
	case '[', '{':
		depth := 0
		for s.i < len(s.b) {
			switch s.b[s.i] {
			case '"':
				if _, err := s.str(); err != nil {
					return err
				}
				continue
			case '[', '{':
				depth++
			case ']', '}':
				depth--
				if depth == 0 {
					s.i++
					return nil
				}
			}
			s.i++
		}
		return errNotTable
	default:
		start := s.i
		for s.i < len(s.b) {
			switch s.b[s.i] {
			case ',', '}', ']', ' ', '\t', '\n', '\r':
				if s.i == start {
					return errNotTable
				}
				return nil
			}
			s.i++
		}
		return errNotTable
	}
}
