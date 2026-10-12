// recordsLen · then the records block, in directory order:
package oraclebin

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
)

const (
	magic = "GOPOBORB"

	version = 2
)

type Record struct {
	Build       string             `json:"build"`
	TreeSources int                `json:"treeSources"`
	Output      map[string]float64 `json:"output"`
	MainHand    map[string]float64 `json:"outputMainHand"`
	OffHand     map[string]float64 `json:"outputOffHand"`
	EnemyOutput map[string]float64 `json:"enemyOutput"`
	Minion      map[string]float64 `json:"minionOutput"`
	MinionMain  map[string]float64 `json:"minionOutputMainHand"`
	MinionOff   map[string]float64 `json:"minionOutputOffHand"`
}

func (r *Record) tables() []*map[string]float64 {
	return []*map[string]float64{
		&r.Output, &r.MainHand, &r.OffHand, &r.EnemyOutput,
		&r.Minion, &r.MinionMain, &r.MinionOff,
	}
}

func Encode(w io.Writer, recs map[string]Record, srcDigest [sha256.Size]byte) error {
	keyIdx := map[string]uint64{}
	valCount := map[uint64]int{}
	for _, rec := range recs {
		for _, tp := range rec.tables() {
			for k, v := range *tp {
				keyIdx[k] = 0
				valCount[math.Float64bits(v)]++
			}
		}
	}

	keys := make([]string, 0, len(keyIdx))
	for k := range keyIdx {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		keyIdx[k] = uint64(i)
	}

	vals := make([]uint64, 0, len(valCount))
	for b := range valCount {
		vals = append(vals, b)
	}
	sort.Slice(vals, func(i, j int) bool {
		if ci, cj := valCount[vals[i]], valCount[vals[j]]; ci != cj {
			return ci > cj
		}
		return vals[i] < vals[j]
	})
	valIdx := make(map[uint64]uint64, len(vals))
	for i, b := range vals {
		valIdx[b] = uint64(i)
	}

	builds := make([]string, 0, len(recs))
	for b := range recs {
		builds = append(builds, b)
	}
	sort.Strings(builds)

	recs2 := make([]byte, 0, 1<<22)
	offsets := make([]uint64, len(builds))
	ids := make([]uint64, 0, 1024)
	for bi, b := range builds {
		offsets[bi] = uint64(len(recs2))
		rec := recs[b]
		recs2 = binary.AppendVarint(recs2, int64(rec.TreeSources))
		for _, tp := range rec.tables() {
			t := *tp
			if t == nil {
				recs2 = append(recs2, 0)
				continue
			}
			recs2 = binary.AppendUvarint(recs2, uint64(len(t))+1)
			ids = ids[:0]
			for k := range t {
				ids = append(ids, keyIdx[k])
			}
			sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
			prev := uint64(0)
			for _, id := range ids {
				recs2 = binary.AppendUvarint(recs2, id-prev)
				prev = id
				recs2 = binary.AppendUvarint(recs2, valIdx[math.Float64bits(t[keys[id]])])
			}
		}
	}

	buf := make([]byte, 0, 1<<20)
	buf = append(buf, magic...)
	buf = append(buf, version)
	buf = append(buf, srcDigest[:]...)
	buf = binary.AppendUvarint(buf, uint64(len(keys)))
	for _, k := range keys {
		buf = binary.AppendUvarint(buf, uint64(len(k)))
		buf = append(buf, k...)
	}
	buf = binary.AppendUvarint(buf, uint64(len(vals)))
	for _, b := range vals {
		buf = binary.LittleEndian.AppendUint64(buf, b)
	}
	buf = binary.AppendUvarint(buf, uint64(len(builds)))
	for bi, b := range builds {
		buf = binary.AppendUvarint(buf, uint64(len(b)))
		buf = append(buf, b...)
		buf = binary.AppendUvarint(buf, offsets[bi])
	}
	buf = binary.AppendUvarint(buf, uint64(len(recs2)))
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("oraclebin: write: %w", err)
	}
	if _, err := w.Write(recs2); err != nil {
		return fmt.Errorf("oraclebin: write: %w", err)
	}
	return nil
}

func EncodeFile(path string, recs map[string]Record, srcDigest [sha256.Size]byte) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp) //nolint:gosec // the path is a build-tooling argument
	if err != nil {
		return fmt.Errorf("oraclebin: create %s: %w", tmp, err)
	}
	if err := Encode(f, recs, srcDigest); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("oraclebin: close %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("oraclebin: rename %s: %w", tmp, err)
	}
	return nil
}

type reader struct {
	b []byte
	i int
}

func (r *reader) uvarint(what string) (uint64, error) {
	v, n := binary.Uvarint(r.b[r.i:])
	if n <= 0 {
		return 0, fmt.Errorf("oraclebin: truncated %s at offset %d", what, r.i)
	}
	r.i += n
	return v, nil
}

func (r *reader) bytes(n uint64, what string) ([]byte, error) {
	if n > uint64(len(r.b)-r.i) { //nolint:gosec // len-r.i is non-negative by the reader invariant
		return nil, fmt.Errorf("oraclebin: truncated %s at offset %d", what, r.i)
	}
	out := r.b[r.i : r.i+int(n)] //nolint:gosec // n <= len(r.b)-r.i, checked above
	r.i += int(n)                //nolint:gosec // same bound
	return out, nil
}

type blob struct {
	keys    []string
	vals    []float64
	names   []string
	offsets []uint64
	recs    []byte
	digest  [sha256.Size]byte
}

func parseHeader(b []byte) (*blob, error) {
	if len(b) < len(magic)+1+sha256.Size {
		return nil, fmt.Errorf("oraclebin: file is %d bytes — is Git LFS checked out?", len(b))
	}
	if string(b[:len(magic)]) != magic {
		return nil, fmt.Errorf("oraclebin: bad magic %q — is Git LFS checked out?", b[:len(magic)])
	}
	if b[len(magic)] != version {
		return nil, fmt.Errorf("oraclebin: version %d, want %d — regenerate with `task oracle-bin`",
			b[len(magic)], version)
	}
	bl := &blob{}
	copy(bl.digest[:], b[len(magic)+1:])
	r := &reader{b: b, i: len(magic) + 1 + sha256.Size}

	nKeys, err := r.uvarint("key count")
	if err != nil {
		return nil, err
	}
	bl.keys = make([]string, nKeys)
	for i := range bl.keys {
		n, err := r.uvarint("key length")
		if err != nil {
			return nil, err
		}
		raw, err := r.bytes(n, "key")
		if err != nil {
			return nil, err
		}
		bl.keys[i] = string(raw)
	}

	nVals, err := r.uvarint("value count")
	if err != nil {
		return nil, err
	}
	if nVals > uint64(len(b)) {
		return nil, fmt.Errorf("oraclebin: value count %d exceeds the file", nVals)
	}
	raw, err := r.bytes(nVals*8, "value dictionary")
	if err != nil {
		return nil, err
	}
	bl.vals = make([]float64, nVals)
	for i := range bl.vals {
		bl.vals[i] = math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8:]))
	}

	nBuilds, err := r.uvarint("build count")
	if err != nil {
		return nil, err
	}
	if nBuilds > uint64(len(b)) {
		return nil, fmt.Errorf("oraclebin: build count %d exceeds the file", nBuilds)
	}
	bl.names = make([]string, nBuilds)
	bl.offsets = make([]uint64, nBuilds)
	for i := range bl.names {
		n, err := r.uvarint("build name length")
		if err != nil {
			return nil, err
		}
		nameRaw, err := r.bytes(n, "build name")
		if err != nil {
			return nil, err
		}
		bl.names[i] = string(nameRaw)
		if bl.offsets[i], err = r.uvarint("build offset"); err != nil {
			return nil, err
		}
	}

	recsLen, err := r.uvarint("records length")
	if err != nil {
		return nil, err
	}
	bl.recs, err = r.bytes(recsLen, "records block")
	if err != nil {
		return nil, err
	}
	if len(bl.names) == 0 {
		return nil, fmt.Errorf("oraclebin: empty oracle — is Git LFS checked out?")
	}
	return bl, nil
}

func (bl *blob) record(i int) (Record, error) {
	rec := Record{Build: bl.names[i]}
	if bl.offsets[i] > uint64(len(bl.recs)) {
		return rec, fmt.Errorf("oraclebin: %s: offset %d past the records block", rec.Build, bl.offsets[i])
	}
	r := &reader{b: bl.recs, i: int(bl.offsets[i])} //nolint:gosec // bounded above
	ts, nn := binary.Varint(r.b[r.i:])
	if nn <= 0 {
		return rec, fmt.Errorf("oraclebin: %s: truncated treeSources", rec.Build)
	}
	r.i += nn
	rec.TreeSources = int(ts)
	for _, tp := range rec.tables() {
		tag, err := r.uvarint("table tag")
		if err != nil {
			return rec, err
		}
		if tag == 0 {
			continue
		}
		count := tag - 1
		if count > uint64(len(r.b)-r.i) { //nolint:gosec // non-negative by the reader invariant
			return rec, fmt.Errorf("oraclebin: table of %d entries exceeds the records block", count)
		}
		t := make(map[string]float64, count)
		id := uint64(0)
		for j := uint64(0); j < count; j++ {
			d, err := r.uvarint("key delta")
			if err != nil {
				return rec, err
			}
			id += d
			vi, err := r.uvarint("value index")
			if err != nil {
				return rec, err
			}
			if id >= uint64(len(bl.keys)) || vi >= uint64(len(bl.vals)) {
				return rec, fmt.Errorf("oraclebin: index out of range (key %d/%d, value %d/%d)",
					id, len(bl.keys), vi, len(bl.vals))
			}
			t[bl.keys[id]] = bl.vals[vi]
		}
		*tp = t
	}
	return rec, nil
}

func Decode(b []byte) (map[string]Record, [sha256.Size]byte, error) {
	bl, err := parseHeader(b)
	if err != nil {
		var zero [sha256.Size]byte
		return nil, zero, err
	}
	out := make(map[string]Record, len(bl.names))
	for i := range bl.names {
		rec, err := bl.record(i)
		if err != nil {
			return nil, bl.digest, err
		}
		out[rec.Build] = rec
	}
	return out, bl.digest, nil
}

func DecodeSubset(b []byte, want []string) (map[string]Record, [sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	bl, err := parseHeader(b)
	if err != nil {
		return nil, zero, err
	}
	index := make(map[string]int, len(bl.names))
	for i, n := range bl.names {
		index[n] = i
	}
	out := make(map[string]Record, len(want))
	var missing []string
	for _, w := range want {
		i, ok := index[w]
		if !ok {
			missing = append(missing, w)
			continue
		}
		rec, err := bl.record(i)
		if err != nil {
			return nil, bl.digest, err
		}
		out[rec.Build] = rec
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, bl.digest, &MissingBuildsError{Missing: missing, Scanned: len(bl.names)}
	}
	return out, bl.digest, nil
}

type MissingBuildsError struct {
	Missing []string
	Scanned int
}

func (e *MissingBuildsError) Error() string {
	return fmt.Sprintf("REFUSING: the oracle blob has no record for %s (scanned %d records). "+
		"A build absent from the blob is not a build with an empty record — pick a name from the "+
		"corpus directory, suffix included", strings.Join(e.Missing, ", "), e.Scanned)
}

func DecodeFile(path string) (map[string]Record, [sha256.Size]byte, error) {
	b, err := os.ReadFile(path) //nolint:gosec // the path is testdata, chosen by the caller
	if err != nil {
		return nil, [sha256.Size]byte{}, fmt.Errorf("oraclebin: read %s: %w", path, err)
	}
	return Decode(b)
}

func SourcePath(binPath string) string {
	return strings.TrimSuffix(binPath, BinExt) + ".jsonl.gz"
}

func BinPath(jsonlPath string) string {
	return strings.TrimSuffix(jsonlPath, ".jsonl.gz") + BinExt
}

const BinExt = ".orb"

func DigestFile(path string) ([sha256.Size]byte, error) {
	var d [sha256.Size]byte
	f, err := os.Open(path) //nolint:gosec // the path is testdata, chosen by the caller
	if err != nil {
		return d, fmt.Errorf("oraclebin: open %s: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return d, fmt.Errorf("oraclebin: hash %s: %w", path, err)
	}
	copy(d[:], h.Sum(nil))
	return d, nil
}

func DecodeJSONL(path string) (map[string]Record, error) {
	f, err := os.Open(path) //nolint:gosec // the path is testdata, chosen by the caller
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("%s: not gzip — is Git LFS checked out?: %w", path, err)
	}
	defer gz.Close()

	out := map[string]Record{}
	scanner := bufio.NewScanner(gz)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out[rec.Build] = rec
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: empty oracle — is Git LFS checked out?", path)
	}
	return out, nil
}

func Load(jsonlPath string) (map[string]Record, error) {
	binPath := BinPath(jsonlPath)
	if recs, ok := loadSideCar(binPath, jsonlPath); ok {
		return recs, nil
	}
	return DecodeJSONL(jsonlPath)
}

func loadSideCar(binPath, jsonlPath string) (map[string]Record, bool) {
	b, err := os.ReadFile(binPath) //nolint:gosec // the path is testdata, chosen by the caller
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "oraclebin: %v — falling back to %s\n", err, jsonlPath)
		}
		return nil, false
	}
	recs, digest, err := Decode(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oraclebin: %v — falling back to %s\n", err, jsonlPath)
		return nil, false
	}
	want, err := DigestFile(jsonlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oraclebin: %v — falling back to %s\n", err, jsonlPath)
		return nil, false
	}
	if digest != want {
		fmt.Fprintf(os.Stderr,
			"oraclebin: %s is STALE (built from %x, blob is %x) — falling back to %s; run `task oracle-bin`\n",
			binPath, digest[:6], want[:6], jsonlPath)
		return nil, false
	}
	return recs, true
}

func Regenerate(jsonlPath string) (string, int, error) {
	recs, err := DecodeJSONL(jsonlPath)
	if err != nil {
		return "", 0, err
	}
	digest, err := DigestFile(jsonlPath)
	if err != nil {
		return "", 0, err
	}
	binPath := BinPath(jsonlPath)
	if err := EncodeFile(binPath, recs, digest); err != nil {
		return "", 0, err
	}
	return binPath, len(recs), nil
}

func LoadSubset(jsonlPath string, want []string) (map[string]Record, error) {
	if len(want) == 0 {
		return nil, fmt.Errorf("REFUSING: LoadSubset was asked for no builds; " +
			"use Load when you want the whole blob")
	}
	for _, w := range want {
		if w == "" {
			return nil, fmt.Errorf("REFUSING: LoadSubset was asked for an empty build name")
		}
	}

	binPath := BinPath(jsonlPath)
	if b, err := os.ReadFile(binPath); err == nil { //nolint:gosec // the path is testdata
		recs, digest, decErr := DecodeSubset(b, want)
		var missing *MissingBuildsError
		switch {
		case errors.As(decErr, &missing):
			if wantDigest, err := DigestFile(jsonlPath); err == nil && digest == wantDigest {
				return nil, decErr
			}
		case decErr != nil:
			fmt.Fprintf(os.Stderr, "oraclebin: %v — falling back to %s\n", decErr, jsonlPath)
		default:
			if wantDigest, err := DigestFile(jsonlPath); err == nil && digest == wantDigest {
				return recs, nil
			}
			fmt.Fprintf(os.Stderr,
				"oraclebin: %s is STALE — falling back to %s; run `task oracle-bin`\n",
				binPath, jsonlPath)
		}
	}
	return decodeJSONLSubset(jsonlPath, want)
}

func decodeJSONLSubset(path string, want []string) (map[string]Record, error) {
	wanted := make(map[string]bool, len(want))
	for _, w := range want {
		wanted[w] = true
	}

	f, err := os.Open(path) //nolint:gosec // the path is testdata, chosen by the caller
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("%s: not gzip — is Git LFS checked out?: %w", path, err)
	}
	defer gz.Close()

	out := make(map[string]Record, len(wanted))
	lines := 0
	scanner := bufio.NewScanner(gz)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		lines++
		name, err := lineBuild(line)
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, lines, err)
		}
		if !wanted[name] {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out[rec.Build] = rec
		if len(out) == len(wanted) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if lines == 0 {
		return nil, fmt.Errorf("%s: empty oracle — is Git LFS checked out?", path)
	}
	if len(out) != len(wanted) {
		missing := make([]string, 0, len(wanted)-len(out))
		for w := range wanted {
			if _, ok := out[w]; !ok {
				missing = append(missing, w)
			}
		}
		sort.Strings(missing)
		return nil, &MissingBuildsError{Missing: missing, Scanned: lines}
	}
	return out, nil
}

func lineBuild(line []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(line))
	tok, err := dec.Token()
	if err != nil {
		return "", fmt.Errorf("not a JSON object: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return "", fmt.Errorf("not a JSON object (starts with %v)", tok)
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("reading object key: %w", err)
		}
		k, ok := key.(string)
		if !ok {
			return "", fmt.Errorf("object key is not a string: %v", key)
		}
		if k != "build" {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return "", fmt.Errorf("skipping value of key %q: %w", k, err)
			}
			continue
		}
		val, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf(`reading the "build" value: %w`, err)
		}
		name, ok := val.(string)
		if !ok {
			return "", fmt.Errorf(`"build" is not a string: %v`, val)
		}
		return name, nil
	}
	return "", fmt.Errorf(`record carries no "build" field`)
}
