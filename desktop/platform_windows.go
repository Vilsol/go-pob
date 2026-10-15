package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func (d *desktop) platform(js string) {
	d.window.OnWindowEvent(events.Windows.WebViewNavigationCompleted, func(*application.WindowEvent) {
		d.window.HandleMessage("wails:runtime:ready")
		if js != "" {
			d.window.ExecJS(js)
		}
	})
}

func platformMenu(*application.Menu) {}
