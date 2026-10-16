package main

import (
	"encoding/base64"
	"log"
	"os"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"

	"github.com/Vilsol/go-pob/desktop/host"
)

var (
	version         = "0.0.0-dev"
	updateURL       string
	updatePublicKey string
	updateManager   string
)

func (d *desktop) updates() *host.Updates {
	if updateManager != "" || updateURL == "" || updatePublicKey == "" {
		return &host.Updates{Manager: updateManager}
	}
	key, err := base64.StdEncoding.DecodeString(updatePublicKey)
	if err != nil {
		log.Printf("updater: public key: %v", err)
		return &host.Updates{}
	}
	p, err := endpoint.New(endpoint.Config{URL: updateURL})
	if err != nil {
		log.Printf("updater: %v", err)
		return &host.Updates{}
	}
	err = d.app.Updater.Init(updater.Config{
		CurrentVersion: version,
		Providers:      []updater.Provider{host.SignedOnly{Provider: p}},
		PublicKey:      key,
		Window:         updater.WindowNone,
	})
	if err != nil {
		log.Printf("updater: %v", err)
		return &host.Updates{}
	}
	return &host.Updates{Updater: d.app.Updater, CanRestart: canReplaceSelf()}
}

func canReplaceSelf() bool {
	if host.AppImagePath() != "" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	for _, dir := range host.ReplaceDirs(exe, runtime.GOOS) {
		f, err := os.CreateTemp(dir, ".go-pob-update-*")
		if err != nil {
			return false
		}
		_ = f.Close()
		_ = os.Remove(f.Name())
	}
	return true
}
