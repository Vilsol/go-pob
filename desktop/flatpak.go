package main

import "github.com/Vilsol/go-pob/desktop/host"

func (d *desktop) nativeDialogs() dialogs {
	if host.FlatpakID() != "" {
		return &host.Portal{}
	}
	return wailsDialogs{d}
}

func uniqueID() string {
	if id := host.FlatpakID(); id != "" {
		return id
	}
	return "io.github.Vilsol.go-pob"
}

func programName() string {
	if id := host.FlatpakID(); id != "" {
		return id
	}
	return "go-pob"
}
