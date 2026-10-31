//go:build js

package main

import (
	"fmt"

	_ "github.com/Vilsol/go-pob/wasm2/exposition"
)

func main() {
	fmt.Println("go-pob2 initialized")
	select {}
}
