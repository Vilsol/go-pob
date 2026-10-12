//go:build js

package main

import (
	"fmt"

	_ "github.com/Vilsol/go-pob/wasm/exposition"
)

func main() {
	fmt.Println("go-pob initialized")
	select {}
}
