//go:build js

package main

import (
	"fmt"

	"github.com/Vilsol/go-pob/wasm/exposition"
)

func main() {
	exposition.Expose()
	fmt.Println("go-pob initialized")
	select {}
}
