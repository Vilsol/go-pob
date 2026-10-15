//go:build !windows && !darwin

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func (d *desktop) platform(string) {}

func platformMenu(*application.Menu) {}
