package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type wailsDialogs struct{ d *desktop }

func (w wailsDialogs) SaveFile(suggested string) (string, error) {
	dlg := w.d.app.Dialog.SaveFile().SetFilename(suggested).CanCreateDirectories(true).AttachToWindow(w.d.window)
	if ext := filepath.Ext(suggested); ext != "" {
		dlg.AddFilter(strings.ToUpper(ext[1:])+" files", "*"+ext)
	}
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("save dialog: %w", err)
	}
	return path, nil
}

func (w wailsDialogs) OpenBuild() (string, error) {
	path, err := w.d.app.Dialog.OpenFile().
		SetTitle("Open build file").
		CanChooseFiles(true).
		AddFilter("Path of Building builds", "*.xml").
		AddFilter("Build codes", "*.txt").
		AttachToWindow(w.d.window).
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("open dialog: %w", err)
	}
	return path, nil
}

func (w wailsDialogs) ChooseFolder(current string) (string, error) {
	path, err := w.d.app.Dialog.OpenFile().
		SetTitle("Builds folder").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		CanCreateDirectories(true).
		SetDirectory(current).
		AttachToWindow(w.d.window).
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("folder dialog: %w", err)
	}
	return path, nil
}
