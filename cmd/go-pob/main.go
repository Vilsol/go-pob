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
	"github.com/Vilsol/go-pob/mcp/prices"
)

func usage() {
	fmt.Fprintln(os.Stderr, `usage: go-pob mcp [--export-dir <dir>] [--prices <snapshot file> | --prices-url <url> | --no-prices]

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
	exportDir := fs.String("export-dir", os.Getenv(mcp.ExportDirEnv), "write one PoB XML file per export_build call without a path here (default $"+mcp.ExportDirEnv+")")
	pricesFile := fs.String("prices", "", "price builds from this poe.ninja snapshot file and never fetch")
	pricesURL := fs.String("prices-url", prices.DefaultURL, "fetch poe.ninja prices through this price service on first need")
	noPrices := fs.Bool("no-prices", false, "disable price_build")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	config.InitLogging(false)
	urlSet := false
	fs.Visit(func(f *flag.Flag) { urlSet = urlSet || f.Name == "prices-url" })
	var book *prices.Book
	if !*noPrices || *pricesFile != "" || urlSet {
		var err error
		if book, err = priceBook(*pricesFile, *pricesURL, urlSet, *noPrices); err != nil {
			fmt.Fprintln(os.Stderr, "mcp:", err)
			return 2
		}
	}
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := mcp.New(mcp.Options{ExportDir: *exportDir, Version: version, Prices: book}).Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "mcp:", err)
		return 1
	}
	return 0
}

func priceBook(file, serviceURL string, urlSet, disabled bool) (*prices.Book, error) {
	switch {
	case disabled && (file != "" || urlSet):
		return nil, fmt.Errorf("--no-prices excludes --prices and --prices-url")
	case file != "" && urlSet:
		return nil, fmt.Errorf("--prices and --prices-url exclude each other")
	case file == "":
		return prices.NewLive("", serviceURL)
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("price snapshot: %w", err)
	}
	defer f.Close()
	return prices.FromSnapshot(f)
}
