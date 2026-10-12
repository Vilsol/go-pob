package parsecache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

const benchSample = 200

func sampleXML(tb testing.TB) [][]byte {
	tb.Helper()
	r, err := checkoutRoot()
	if err != nil {
		tb.Skipf("%v", err)
	}
	dir := filepath.Join(r, "testdata", "corpus-"+raw.LatestVersion)
	ents, err := os.ReadDir(dir)
	if err != nil {
		tb.Skipf("no corpus at %s: %v", dir, err)
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if filepath.Ext(e.Name()) == ".xml" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) < benchSample {
		tb.Skipf("corpus has %d builds, need %d", len(names), benchSample)
	}
	out := make([][]byte, 0, benchSample)
	for _, n := range names[:benchSample] {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			tb.Fatalf("REFUSING: %v", err)
		}
		out = append(out, b)
	}
	return out
}

func sampleParsed(tb testing.TB) []*pob.PathOfBuilding {
	tb.Helper()
	xmls := sampleXML(tb)
	out := make([]*pob.PathOfBuilding, 0, len(xmls))
	for _, x := range xmls {
		b, err := builds.ParseBuild(x)
		if err != nil {
			tb.Fatalf("REFUSING: %v", err)
		}
		out = append(out, b)
	}
	return out
}

// BenchmarkParseBuild is the cost the cache exists to avoid.
func BenchmarkParseBuild(b *testing.B) {
	xmls := sampleXML(b)
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if _, err := builds.ParseBuild(xmls[i%len(xmls)]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGobEncode(b *testing.B) {
	parsed := sampleParsed(b)
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(parsed[i%len(parsed)]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGobDecode(b *testing.B) {
	parsed := sampleParsed(b)
	blobs := make([][]byte, len(parsed))
	for i, p := range parsed {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(p); err != nil {
			b.Fatal(err)
		}
		blobs[i] = buf.Bytes()
	}
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		var out pob.PathOfBuilding
		if err := gob.NewDecoder(bytes.NewReader(blobs[i%len(blobs)])).Decode(&out); err != nil {
			b.Fatal(err)
		}
	}
}

func jsonEncodable(tb testing.TB) []*pob.PathOfBuilding {
	tb.Helper()
	var out []*pob.PathOfBuilding
	for _, p := range sampleParsed(tb) {
		if _, err := json.Marshal(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func BenchmarkJSONEncode(b *testing.B) {
	parsed := jsonEncodable(b)
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if _, err := json.Marshal(parsed[i%len(parsed)]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONDecode(b *testing.B) {
	parsed := jsonEncodable(b)
	blobs := make([][]byte, len(parsed))
	for i, p := range parsed {
		x, err := json.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		blobs[i] = x
	}
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		var out pob.PathOfBuilding
		if err := json.Unmarshal(blobs[i%len(blobs)], &out); err != nil {
			b.Fatal(err)
		}
	}
}

// TestCodecSizesAndFidelity is the report the choice of format rests on: on-disk bytes per build and, far more importantly, whether the round trip is LOSSLESS.
func TestCodecSizesAndFidelity(t *testing.T) {
	parsed := sampleParsed(t)

	var gobBytes, jsonBytes int
	gobLossy, jsonLossy, jsonRefused := 0, 0, 0
	for _, p := range parsed {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(p); err != nil {
			t.Fatalf("REFUSING: gob encode: %v", err)
		}
		gobBytes += buf.Len()
		var g pob.PathOfBuilding
		if err := gob.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&g); err != nil {
			t.Fatalf("REFUSING: gob decode: %v", err)
		}
		if !reflect.DeepEqual(p, &g) {
			gobLossy++
		}

		jb, err := json.Marshal(p)
		if err != nil {
			jsonRefused++
			continue
		}
		jsonBytes += len(jb)
		var j pob.PathOfBuilding
		if err := json.Unmarshal(jb, &j); err != nil {
			t.Fatalf("REFUSING: json decode: %v", err)
		}
		if !reflect.DeepEqual(p, &j) {
			jsonLossy++
		}
	}

	n := len(parsed)
	t.Logf("over %d builds: gob %d B/build, lossy on %d; json %d B/build, lossy on %d, REFUSED on %d",
		n, gobBytes/n, gobLossy, jsonBytes/max(n-jsonRefused, 1), jsonLossy, jsonRefused)
}

func BenchmarkCacheEncode(b *testing.B) {
	parsed := sampleParsed(b)
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if _, err := Encode(parsed[i%len(parsed)]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheDecode(b *testing.B) {
	parsed := sampleParsed(b)
	blobs := make([][]byte, len(parsed))
	for i, p := range parsed {
		x, err := Encode(p)
		if err != nil {
			b.Fatal(err)
		}
		blobs[i] = x
	}
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		var out pob.PathOfBuilding
		if err := Decode(blobs[i%len(blobs)], &out); err != nil {
			b.Fatal(err)
		}
	}
}

// TestCacheRoundTripIsLossless is the whole justification for the presence bitmap.
func TestCacheRoundTripIsLossless(t *testing.T) {
	parsed := sampleParsed(t)
	total := 0
	for i, p := range parsed {
		blob, err := Encode(p)
		if err != nil {
			t.Fatalf("REFUSING: encode build %d: %v", i, err)
		}
		total += len(blob)
		var got pob.PathOfBuilding
		if err := Decode(blob, &got); err != nil {
			t.Fatalf("REFUSING: decode build %d: %v", i, err)
		}
		if !reflect.DeepEqual(p, &got) {
			t.Fatalf("build %d does not round-trip", i)
		}
	}
	t.Logf("%d builds round-trip losslessly, %d B/build on disk", len(parsed), total/len(parsed))
}

// TestCachedTypeTreeHasNothingTheWalkCannotSee refuses a map.
func TestCachedTypeTreeHasNothingTheWalkCannotSee(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var visit func(path string, ty reflect.Type)
	visit = func(path string, ty reflect.Type) {
		if seen[ty] {
			return
		}
		seen[ty] = true
		switch ty.Kind() {
		case reflect.Map, reflect.Chan, reflect.Func:
			t.Errorf("%s is a %s: the presence walk records no bit for it", path, ty.Kind())
		case reflect.Pointer, reflect.Slice, reflect.Array:
			visit(path+"[]", ty.Elem())
		case reflect.Struct:
			for i := range ty.NumField() {
				f := ty.Field(i)
				if !f.IsExported() {
					continue
				}
				if f.Type.Kind() == reflect.Interface {
					if f.Tag.Get("xml") != "-" {
						t.Errorf("%s.%s is a saved interface field; gob cannot carry it unregistered",
							path, f.Name)
					}
					continue
				}
				visit(path+"."+f.Name, f.Type)
			}
		default:
		}
	}
	visit("PathOfBuilding", reflect.TypeOf(pob.PathOfBuilding{}))
}

// BenchmarkPresenceWalk isolates the ONE thing this codec adds to gob.
func BenchmarkPresenceWalk(b *testing.B) {
	parsed := sampleParsed(b)
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		var bs bitset
		walk(reflect.ValueOf(parsed[i%len(parsed)]).Elem(), &bs, nil)
		if bs.n == 0 {
			b.Fatal("walk recorded nothing")
		}
	}
}
