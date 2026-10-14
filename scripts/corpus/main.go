package main

import (
	"fmt"
	"os"
	"strings"
)

const usageText = `corpus — league-parameterised PoB build corpus harvester

usage: go run ./scripts/corpus <command> [flags]

commands:
  ladder     fetch ladder characters from GGG's own API as importable JSON
  discover   crawl a source and append to the link index
  fetch      resolve indexed links to PoB export codes (resumable)
  validate   decode, apply acceptance criteria, dedup, write builds/
  manifest   render MANIFEST.md from a validated corpus
  snapshot   import a pob.codes test-builds snapshot straight into a corpus directory
  archive    write the DURABLE per-league link index into the repo (winners AND rejects)
  seed       rebuild a working index from a committed archive — costs zero re-crawling
  linkrot    sample an archive's links to measure how many are still reachable
  status     summarise every stage's artifacts under -work
  selftest   prove each acceptance check and each fetch classifier can fail

Every command takes -work <dir> (default $CORPUS_WORK or ./corpus-work).
Run "<command> -h" for that command's flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}

	cmd, args := os.Args[1], os.Args[2:]

	var err error
	switch cmd {
	case "ladder":
		err = cmdLadder(args)
	case "discover":
		err = cmdDiscover(args)
	case "fetch":
		err = cmdFetch(args)
	case "validate":
		err = cmdValidate(args)
	case "manifest":
		err = cmdManifest(args)
	case "snapshot":
		err = cmdSnapshot(args)
	case "archive":
		err = cmdArchive(args)
	case "seed":
		err = cmdSeed(args)
	case "linkrot":
		err = cmdLinkrot(args)
	case "status":
		err = cmdStatus(args)
	case "selftest":
		err = cmdSelftest(args)
	case "-h", "--help", "help":
		fmt.Print(usageText)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usageText)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "\ncorpus %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func banner(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	fmt.Printf("\n%s\n%s\n", line, strings.Repeat("=", len(line)))
}
