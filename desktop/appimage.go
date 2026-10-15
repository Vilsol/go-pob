package main

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Vilsol/go-pob/desktop/host"
)

//go:embed build/linux/go-pob.desktop
var desktopEntry []byte

func (d *desktop) appImageMenu(help *application.Menu) {
	appImage := host.AppImagePath()
	if appImage == "" {
		return
	}
	help.Add("Add to applications menu").OnClick(func(*application.Context) {
		data := os.Getenv("XDG_DATA_HOME")
		if data == "" {
			home, _ := os.UserHomeDir()
			data = filepath.Join(home, ".local", "share")
		}
		notes, err := host.IntegrateAppImage(context.Background(), appImage, data, desktopEntry, icon)
		switch {
		case err != nil:
			d.toast("Add to applications menu failed\n" + err.Error())
		case notes != "":
			d.toast("Added to the applications menu\n" + notes)
		default:
			d.toast("Added to the applications menu; pob:// links now open go-pob")
		}
	})
}
