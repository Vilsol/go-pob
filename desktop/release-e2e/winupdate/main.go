package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"

	"github.com/Vilsol/go-pob/desktop/host"
)

var version = "0.0.0"

type headless struct{}

func (headless) Emit(string, ...any) bool                              { return true }
func (headless) OnEvent(string, func(any)) func()                      { return func() {} }
func (headless) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }
func (headless) Quit()                                                 { logf("quit"); os.Exit(0) }

func logf(format string, args ...any) {
	f, err := os.OpenFile(os.Getenv("WINUPDATE_LOG"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // the proof script names the log
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "winupdate: "+format+"\n", args...)
}

func main() {
	updater.HandleHelperMode()
	logf("running %s", version)
	if len(os.Args) != 4 {
		return
	}
	mode, url, key64 := os.Args[1], os.Args[2], os.Args[3]
	key, err := base64.StdEncoding.DecodeString(key64)
	if err != nil {
		logf("key: %v", err)
		os.Exit(1)
	}
	p, err := endpoint.New(endpoint.Config{URL: url})
	if err != nil {
		logf("endpoint: %v", err)
		os.Exit(1)
	}
	u := updater.New(headless{})
	err = u.Init(updater.Config{CurrentVersion: version, Providers: []updater.Provider{host.SignedOnly{Provider: p}}, PublicKey: key, Window: updater.WindowNone})
	if err != nil {
		logf("init: %v", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	rel, err := u.Check(ctx)
	if err != nil || rel == nil {
		logf("check: %v %v", rel, err)
		os.Exit(1)
	}
	logf("check: version %s, %s", rel.Version, rel.Artifact.Filename)
	if mode == "check" {
		return
	}
	if err := u.DownloadAndInstall(ctx); err != nil {
		logf("install: %v", err)
		os.Exit(1)
	}
	logf("install: ready")
	if mode == "restart" {
		if err := u.Restart(ctx); err != nil {
			logf("restart: %v", err)
			os.Exit(1)
		}
	}
}
