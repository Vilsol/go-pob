package main

import (
	"fmt"
	"os"

	"github.com/Vilsol/go-pob/internal/oraclebin"
)

func oracleBinEmit(args []string) int {
	check := false
	rest := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "--check":
			check = true
		default:
			if len(a) > 2 && a[:2] == "--" {
				fmt.Fprintf(os.Stderr, "oracle-bin: unknown flag %q\n", a)
				return 64
			}
			rest = append(rest, a)
		}
	}
	if len(rest) != 1 {
		fmt.Fprint(os.Stderr, "usage: gopob oracle-bin [--check] <oracle.jsonl.gz>\n")
		return 64
	}
	jsonlPath := rest[0]

	if check {
		binPath := oraclebin.BinPath(jsonlPath)
		b, err := os.ReadFile(binPath) //nolint:gosec // G703: a build-tooling argument, not user input
		if err != nil {
			fmt.Fprintf(os.Stderr, "STALE: %v\n", err)
			return 1
		}
		_, digest, err := oraclebin.Decode(b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "STALE: %v\n", err)
			return 1
		}
		want, err := oraclebin.DigestFile(jsonlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "oracle-bin: %v\n", err)
			return 1
		}
		if digest != want {
			fmt.Fprintf(os.Stderr, "STALE: %s was built from %x, %s is %x — run `task oracle-bin`\n",
				binPath, digest[:8], jsonlPath, want[:8])
			return 1
		}
		fmt.Printf("CURRENT: %s matches %s (%x)\n", binPath, jsonlPath, digest[:8])
		return 0
	}

	binPath, n, err := oraclebin.Regenerate(jsonlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle-bin: %v\n", err)
		return 1
	}
	st, err := os.Stat(binPath) //nolint:gosec // G703: a build-tooling argument, not user input
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle-bin: %v\n", err)
		return 1
	}
	fmt.Printf("wrote %s: %d builds, %d bytes\n", binPath, n, st.Size())
	return 0
}
