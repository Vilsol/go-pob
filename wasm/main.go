//go:build js

package main

import (
	"fmt"

	_ "github.com/Vilsol/go-pob/utils/jsfetch"
	_ "github.com/Vilsol/go-pob/utils/libm"
	_ "github.com/Vilsol/go-pob/wasm/exposition"
)

func main() {
	fmt.Println("go-pob initialized")
	select {}
}
