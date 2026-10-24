package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

var reLuaFlag = regexp.MustCompile(`(?m)^KeywordFlag\.(\w+)\s*=\s*(0x[0-9a-fA-F]+)`)

func hexv(s string) int64 {
	v, _ := strconv.ParseInt(strings.TrimPrefix(s, "0x"), 16, 64)
	return v
}

func kwBitsParity() int {
	if _, err := root.Resolve(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return root.RefusalCode
	}

	pob := pobPinDir()
	lua := filepath.Join(pob, "src/Data/Global.lua")
	luaSrc, readErr := os.ReadFile(lua) //nolint:gosec // an operator-supplied checkout path
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: pinned PoB not readable at %s; set POB_DIR\n", lua)
		return 3
	}
	pobFlags := map[string]int64{}
	for _, m := range reLuaFlag.FindAllStringSubmatch(string(luaSrc), -1) {
		pobFlags[m[1]] = hexv(m[2])
	}
	if len(pobFlags) < 20 {
		fmt.Fprintf(os.Stderr, "REFUSING: parsed only %d flags from %s\n", len(pobFlags), lua)
		return 3
	}

	goFlags := make(map[string]int64, len(mod.KeywordFlagsByName))
	for name, v := range mod.KeywordFlagsByName {
		goFlags[name] = int64(v)
	}

	names := make([]string, 0, len(goFlags))
	for name := range goFlags {
		names = append(names, name)
	}
	sort.Strings(names)

	bad := 0
	for _, name := range names {
		v := goFlags[name]
		if pv, ok := pobFlags[name]; ok && pv != v {
			fmt.Printf("mod/common.go: %s declares 0x%08x, PoB writes 0x%08x\n", name, v, pv)
			bad++
		}
	}

	if bad > 0 {
		fmt.Fprintf(os.Stderr, "\n%d mismatch(es). The emitters decode numbers PoB WROTE into names "+
			"taken from mod/common.go, so mod/common.go must agree with PoB or every generated name "+
			"means something else.\n", bad)
		return 1
	}
	fmt.Printf("KeywordFlag parity OK: mod/common.go (%d flags) vs the pinned PoB (%d flags)\n",
		len(goFlags), len(pobFlags))
	return 0
}

func errnoText(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "No such file or directory"
	case errors.Is(err, fs.ErrPermission):
		return "Permission denied"
	case errors.Is(err, syscall.EISDIR):
		return "Is a directory"
	case errors.Is(err, syscall.ENOTDIR):
		return "Not a directory"
	default:
		return err.Error()
	}
}
