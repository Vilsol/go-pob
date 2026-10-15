package main

import (
	"net/url"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func (d *desktop) platform(string) {
	var loaded atomic.Bool
	d.window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) { loaded.Store(true) })
	open := func(text string) {
		if loaded.Load() {
			d.importText(text)
		} else {
			d.window.SetURL("/?" + url.Values{"import": {text}}.Encode())
		}
		d.window.Show()
		d.window.Focus()
	}
	d.app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(e *application.ApplicationEvent) {
		open(e.Context().URL())
	})
	d.app.Event.OnApplicationEvent(events.Common.ApplicationOpenedWithFile, func(e *application.ApplicationEvent) {
		text, err := d.files.Open(e.Context().Filename())
		if err != nil {
			d.toast("Open build file failed\n" + err.Error())
			return
		}
		open(text)
	})
}

func platformMenu(menu *application.Menu) {
	menu.AddRole(application.AppMenu)
}
