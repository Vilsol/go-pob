// Package mcp serves go-pob's calculation engine to AI assistants over the Model Context Protocol.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Vilsol/go-pob/data/raw"
)

// ExportDirEnv names the directory export_build writes one PoB XML file per export into.
const ExportDirEnv = "GOPOB_MCP_EXPORT_DIR"

type Options struct {
	ExportDir string
	Version   string
}

type Server struct {
	opts Options

	mu              sync.Mutex
	handles         map[string]*handle
	order           []string
	nextHandle      int
	pendingWarnings []string
	exportSeq       int

	ready   atomic.Bool
	loadErr atomic.Value
}

func New(opts Options) *Server {
	return &Server{opts: opts, handles: map[string]*handle{}}
}

// LoadData loads the game data; tool calls before it finishes answer DATA_LOADING.
func (s *Server) LoadData() error {
	err := raw.InitializeAll(raw.LatestVersion, nil)
	if err == nil {
		err = raw.LoadDeferred()
	}
	if err != nil {
		s.loadErr.Store(err)
		return err
	}
	s.ready.Store(true)
	return nil
}

type toolFunc func(s *Server, args json.RawMessage) (any, *ToolError)

type toolDef struct {
	name   string
	desc   string
	schema string
	run    toolFunc
}

func (s *Server) SDKServer() *sdk.Server {
	version := s.opts.Version
	if version == "" {
		version = "dev"
	}
	srv := sdk.NewServer(&sdk.Implementation{Name: "go-pob", Version: version}, &sdk.ServerOptions{
		Instructions: "Path of Building (PoE1) calculations. Start with new_build or import_build; every edit returns a stat delta; dry_run:true previews.",
	})
	for _, t := range tools() {
		srv.AddTool(&sdk.Tool{Name: t.name, Description: t.desc, InputSchema: json.RawMessage(t.schema)}, s.handler(t))
	}
	return srv
}

// Run serves MCP over stdin/stdout until the client disconnects.
func (s *Server) Run(ctx context.Context) error {
	go func() { _ = s.LoadData() }() //nolint:contextcheck // data loading outlives no request and takes no context
	if err := s.SDKServer().Run(ctx, &sdk.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}

func (s *Server) handler(t toolDef) sdk.ToolHandler {
	return func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		out, terr := s.call(t, req.Params.Arguments) //nolint:contextcheck // the engine takes no context; a tool call runs to completion
		if terr != nil {
			body, _ := json.Marshal(map[string]any{"error": terr})
			return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: string(body)}}}, nil
		}
		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: string(out)}}}, nil
	}
}

// call runs one tool under the server lock; tools never run concurrently, so the shared calculator caches stay
// single-threaded.
func (s *Server) call(t toolDef, args json.RawMessage) ([]byte, *ToolError) {
	if err, ok := s.loadErr.Load().(error); ok && err != nil {
		return nil, errf(CodeInternal, "game data failed to load: %v", err)
	}
	if !s.ready.Load() {
		e := errf(CodeDataLoading, "Game data is still loading.")
		e.RetryAfterMs = 3000
		e.Hint = "Retry the same call in a few seconds."
		return nil, e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(args) == 0 || string(args) == "null" {
		args = json.RawMessage("{}")
	}
	var res any
	var terr *ToolError
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "panic in %s: %v\n%s", t.name, r, debug.Stack())
				res = nil
				terr = errf(CodeInternal, "engine panic: %v", r)
				terr.Hint = "The build was not changed."
			}
		}()
		res, terr = t.run(s, args)
	}()
	if terr != nil {
		return nil, terr
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, errf(CodeInternal, "result does not marshal: %v", err)
	}
	if len(s.pendingWarnings) > 0 && len(b) > 0 && b[0] == '{' {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			m["server_warnings"] = s.pendingWarnings
			s.pendingWarnings = nil
			b, _ = json.Marshal(m)
		}
	}
	return b, nil
}

// decode reads args into v, refusing properties the schema does not list.
func decode(args json.RawMessage, schema string, v any) *ToolError {
	d := json.NewDecoder(bytes.NewReader(args))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		msg := err.Error()
		var ute *json.UnmarshalTypeError
		if errors.As(err, &ute) {
			msg = fmt.Sprintf("%s must be %s, not %s", ute.Field, ute.Type.String(), ute.Value)
		}
		return invalidArg("Invalid arguments: "+msg, schemaProps(schema))
	}
	return nil
}

// schemaProps lists the schema's properties, and the item properties of its array-of-object properties as "ops[].x".
func schemaProps(schema string) []string {
	var sc struct {
		Properties map[string]struct {
			Items struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"items"`
		} `json:"properties"`
	}
	_ = json.Unmarshal([]byte(schema), &sc)
	var keys []string
	for k, p := range sc.Properties {
		keys = append(keys, k)
		for ik := range p.Items.Properties {
			keys = append(keys, k+"[]."+ik)
		}
	}
	sort.Strings(keys)
	return keys
}
