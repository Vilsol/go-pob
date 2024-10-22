//go:build tools

package main

import (
	"os"

	"github.com/Vilsol/go-pob/wasm/exposition"
)

//go:generate go run tools.go types

func main() {
	if len(os.Args) < 2 {
		return
	}

	switch os.Args[1] {
	case "types":
		generateTypes()
	}
}

func generateTypes() {
	e := exposition.Expose()
	tsFile, jsFile, err := e.Build()
	if err != nil {
		panic(err)
	}

	tsFile = "/* eslint-disable */\n" + tsFile
	jsFile = "/* eslint-disable */\n// @ts-nocheck\n" + jsFile

	if err := os.MkdirAll("./frontend/src/lib/types", 0777); err != nil {
		if !os.IsExist(err) {
			panic(err)
		}
	}

	if err := os.WriteFile("./frontend/src/lib/types/index.js", []byte(jsFile), 0777); err != nil {
		panic(err)
	}

	if err := os.WriteFile("./frontend/src/lib/types/index.d.ts", []byte(tsFile), 0777); err != nil {
		panic(err)
	}
}
