// Command go-pob runs go-pob's calculation engine outside the browser.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/mcp"
)

func usage() {
	fmt.Fprintln(os.Stderr, `usage: go-pob mcp [--export-dir <dir>]

  mcp   serve the Model Context Protocol over stdin/stdout for AI assistants`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "mcp":
		os.Exit(runMCP(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runMCP(args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	exportDir := fs.String("export-dir", os.Getenv(mcp.ExportDirEnv), "write one PoB XML file per export_build call here (default $"+mcp.ExportDirEnv+")")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	config.InitLogging(false)
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := mcp.New(mcp.Options{ExportDir: *exportDir, Version: version}).Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "mcp:", err)
		return 1
	}
	return 0
}
