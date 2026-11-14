package mcp

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
)

var loadOnce sync.Once

func testServer(t *testing.T) *Server {
	t.Helper()
	s := New(Options{ExportDir: t.TempDir()})
	var err error
	loadOnce.Do(func() { err = s.LoadData() })
	if err != nil {
		t.Fatal(err)
	}
	s.ready.Store(true)
	return s
}

func toolByName(t *testing.T, name string) toolDef {
	t.Helper()
	for _, d := range tools() {
		if d.name == name {
			return d
		}
	}
	t.Fatalf("no tool %s", name)
	return toolDef{}
}

// callRaw runs a tool and returns its JSON body and error.
func callRaw(t *testing.T, s *Server, name string, args any) ([]byte, *ToolError) {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return s.call(toolByName(t, name), b)
}

func call(t *testing.T, s *Server, name string, args any) map[string]any {
	t.Helper()
	out, terr := callRaw(t, s, name, args)
	if terr != nil {
		t.Fatalf("%s: %s %s", name, terr.Code, terr.Message)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func callErr(t *testing.T, s *Server, name string, args any) *ToolError {
	t.Helper()
	_, terr := callRaw(t, s, name, args)
	if terr == nil {
		t.Fatalf("%s: expected an error", name)
	}
	return terr
}

func fixture(name string) string {
	p, _ := filepath.Abs(filepath.Join("..", "testdata", "many-builds", name))
	return p
}

// importFixture imports the shared mid build used by most tests.
func importFixture(t *testing.T, s *Server) string {
	t.Helper()
	res := call(t, s, "import_build", map[string]any{"source": fixture("3.xml")})
	return res["handle"].(string)
}

func stat(m map[string]any, key string) float64 {
	v, _ := m["stats"].(map[string]any)[key].(float64)
	return v
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// assertSize fails when a tool's result exceeds its output budget.
func assertSize(t *testing.T, s *Server, name string, args any, limit int) map[string]any {
	t.Helper()
	out, terr := callRaw(t, s, name, args)
	if terr != nil {
		t.Fatalf("%s: %s %s", name, terr.Code, terr.Message)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	size := len(out)
	if item, ok := m["item"]; ok {
		b, _ := json.Marshal(item)
		size -= len(b)
	}
	if size > limit {
		t.Fatalf("%s returned %d B, budget %d B: %s", name, size, limit, out)
	}
	return m
}
