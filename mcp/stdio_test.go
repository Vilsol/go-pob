package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Vilsol/go-pob/data/raw"
)

// TestStdioEndToEnd builds the go-pob binary and drives `go-pob mcp` over stdin/stdout: initialize, tools/list, then
// a tool call that needs the game data.
func TestStdioEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bin := filepath.Join(t.TempDir(), "go-pob")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "../cmd/go-pob")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	exportDir := t.TempDir()
	cmd := exec.CommandContext(ctx, bin, "mcp")
	cmd.Env = append(os.Environ(), "POE_VERSION="+raw.LatestVersion, ExportDirEnv+"="+exportDir)
	cmd.Stderr = os.Stderr

	client := sdk.NewClient(&sdk.Implementation{Name: "stdio-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if info := session.InitializeResult().ServerInfo; info.Name != "go-pob" {
		t.Fatalf("server = %v", info)
	}

	list, err := session.ListTools(ctx, &sdk.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(list.Tools))
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
	}
	want := []string{"compare_builds", "edit_build", "edit_items", "edit_skills", "edit_tree", "export_build", "get_build",
		"get_stats", "import_build", "new_build", "optimize", "rank_candidates", "revert_build", "search_data"}
	slices.Sort(names)
	if !slices.Equal(names, want) {
		t.Fatalf("tools = %v", names)
	}

	var res *sdk.CallToolResult
	for {
		res, err = session.CallTool(ctx, &sdk.CallToolParams{Name: "new_build", Arguments: map[string]any{"class": "Templar"}})
		if err != nil {
			t.Fatal(err)
		}
		body := res.Content[0].(*sdk.TextContent).Text
		var e struct{ Error *ToolError }
		if res.IsError && json.Unmarshal([]byte(body), &e) == nil && e.Error != nil && e.Error.Code == CodeDataLoading {
			time.Sleep(time.Duration(e.Error.RetryAfterMs/10) * time.Millisecond)
			continue
		}
		if res.IsError {
			t.Fatalf("new_build: %s", body)
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(body), &out); err != nil || out["handle"] != "b1" {
			t.Fatalf("new_build = %s", body)
		}
		break
	}

	res, err = session.CallTool(ctx, &sdk.CallToolParams{Name: "export_build", Arguments: map[string]any{"handle": "b1"}})
	if err != nil || res.IsError {
		t.Fatalf("export_build: %v %v", err, res.Content)
	}
	if files, _ := filepath.Glob(filepath.Join(exportDir, "*.xml")); len(files) != 1 {
		t.Fatalf("export files = %v", files)
	}
	res, err = session.CallTool(ctx, &sdk.CallToolParams{Name: "get_stats", Arguments: map[string]any{"handle": "b9"}})
	if err != nil || !res.IsError {
		t.Fatalf("an unknown handle is not a tool error: %v %v", err, res)
	}
}
