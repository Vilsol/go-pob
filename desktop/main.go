package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Vilsol/go-pob/desktop/host"
)

//go:embed all:dist
var dist embed.FS

//go:embed icon.png
var icon []byte

//go:embed fonts/fonts.css fonts/*.woff2
var fonts embed.FS

func main() {
	pages, err := pageHandler()
	if err != nil {
		log.Fatal(err)
	}

	dirs, err := host.DefaultDirs()
	if err != nil {
		log.Fatal(err)
	}
	d := &desktop{files: &host.Files{}, dirs: dirs, configPath: filepath.Join(dirs.Config, "desktop.json")}
	d.dialogs = probeDialogs(d.nativeDialogs())
	d.files.Dialogs = d.dialogs
	d.resolveBuilds()
	dirs.Builds = func() string { return d.builds().Dir }

	native := http.NewServeMux()
	native.Handle("/fs/", http.StripPrefix("/fs", &host.Store{Roots: dirs.Root, Suffix: map[string]string{"builds": ".xml"}}))
	native.HandleFunc("GET /config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(d.builds())
	})
	native.Handle("/fonts/", http.FileServerFS(fonts))
	native.Handle("/art/", http.StripPrefix("/art", &host.Art{Dir: filepath.Join(filepath.Dir(dirs.Cache), "art"), Upstream: "https://web.poecdn.com/image/"}))
	native.HandleFunc("POST /save", d.files.Save)
	native.HandleFunc("GET /opened/{token}", d.files.Opened)
	native.HandleFunc("POST /open", func(w http.ResponseWriter, _ *http.Request) {
		go d.openBuildFile()
		w.WriteHeader(http.StatusAccepted)
	})
	native.HandleFunc("POST /hint", d.hint)
	native.HandleFunc("POST /builds-folder", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("default") {
			go d.resetBuildsFolder()
		} else {
			go d.chooseBuildsFolder()
		}
		w.WriteHeader(http.StatusAccepted)
	})
	probeJS := probe(native, func() { d.app.Quit() })

	d.app = application.New(application.Options{
		Name:        "go-pob",
		Description: "Path of Building, in Go",
		Icon:        icon,
		Assets: application.AssetOptions{
			Handler:        pages,
			Middleware:     host.Routes(apiHandler(), native),
			DisableLogging: true,
		},
		Linux: application.LinuxOptions{
			ProgramName:   programName(),
			ApplicationID: host.FlatpakID(),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: uniqueID(),
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if text := host.LaunchImport(data.Args[min(1, len(data.Args)):], data.WorkingDir, d.files); text != "" {
					d.importText(text)
				}
				d.window.Restore()
				d.window.Focus()
			},
		},
	})

	d.updates().Register(native)

	start := "/"
	if text := host.LaunchImport(os.Args[1:], "", d.files); text != "" {
		start = "/?" + url.Values{"import": {text}}.Encode()
	}
	d.window = d.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:              "go-pob",
		Width:              1440,
		Height:             900,
		URL:                start,
		UseApplicationMenu: true,
		JS:                 runtimeReady + probeJS,
	})
	d.platform(probeJS)
	d.app.Menu.Set(d.menu())

	if err := d.app.Run(); err != nil {
		log.Fatal(err)
	}
}

const runtimeReady = `window.webkit?.messageHandlers?.external?.postMessage("wails:runtime:ready");`

type desktop struct {
	app        *application.App
	window     *application.WebviewWindow
	files      *host.Files
	dialogs    dialogs
	dirs       host.Dirs
	configPath string

	mu     sync.Mutex
	choice host.BuildsChoice
}

type dialogs interface {
	host.Dialogs
	ChooseFolder(current string) (string, error)
}

func (d *desktop) builds() host.BuildsChoice {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.choice
}

func (d *desktop) resolveBuilds() {
	home, _ := os.UserHomeDir()
	choice := host.ChooseBuilds(host.LoadConfig(d.configPath), host.FindPoBBuilds(home, os.Getenv("WINEPREFIX"), host.UserDocuments(home)), d.dirs.AppBuilds)
	d.mu.Lock()
	d.choice = choice
	d.mu.Unlock()
}

func (d *desktop) chooseBuildsFolder() {
	dir, err := d.dialogs.ChooseFolder(d.builds().Dir)
	if err != nil {
		d.toast("Builds folder\n" + err.Error())
		return
	}
	if dir == "" {
		return
	}
	c := host.LoadConfig(d.configPath)
	c.BuildsDir = dir
	if err := host.SaveConfig(d.configPath, c); err != nil {
		d.toast("Builds folder\n" + err.Error())
		return
	}
	d.resolveBuilds()
	d.dispatch("pob:desktop-reload", "")
}

func (d *desktop) resetBuildsFolder() {
	c := host.LoadConfig(d.configPath)
	c.BuildsDir = ""
	if err := host.SaveConfig(d.configPath, c); err != nil {
		d.toast("Builds folder\n" + err.Error())
		return
	}
	d.resolveBuilds()
	d.dispatch("pob:desktop-reload", "")
}

func (d *desktop) hint(w http.ResponseWriter, _ *http.Request) {
	c := host.LoadConfig(d.configPath)
	text := host.FlatpakHint(c, d.builds(), host.FlatpakID() != "")
	if text == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	c.FlatpakHint = true
	if err := host.SaveConfig(d.configPath, c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(text))
}

func (d *desktop) menu() *application.Menu {
	menu := d.app.NewMenu()
	platformMenu(menu)
	file := menu.AddSubmenu("File")
	file.Add("Open build file…").SetAccelerator("CmdOrCtrl+O").OnClick(func(*application.Context) { d.openBuildFile() })
	file.AddSeparator()
	file.Add("Builds folder…").OnClick(func(*application.Context) { d.dispatch("pob:desktop-builds-folder", "") })
	file.Add("Default builds folder").OnClick(func(*application.Context) { d.dispatch("pob:desktop-builds-folder", "default") })
	file.AddSeparator()
	file.AddRole(application.Quit)
	menu.AddRole(application.EditMenu)
	view := menu.AddSubmenu("View")
	view.Add("Reload").SetAccelerator("CmdOrCtrl+R").OnClick(func(*application.Context) { d.window.Reload() })
	view.AddRole(application.ToggleFullscreen)
	help := menu.AddSubmenu("Help")
	help.Add("About go-pob").OnClick(func(*application.Context) { d.app.Menu.ShowAbout() })
	d.appImageMenu(help)
	return menu
}

func (d *desktop) openBuildFile() {
	path, err := d.dialogs.OpenBuild()
	if err != nil {
		d.toast("Open build file failed\n" + err.Error())
		return
	}
	if path == "" {
		return
	}
	text, err := d.files.Open(path)
	if err != nil {
		d.toast("Open build file failed\n" + err.Error())
		return
	}
	d.importText(text)
}

func (d *desktop) importText(text string) {
	b, _ := json.Marshal(text)
	d.window.ExecJS(fmt.Sprintf(`if (window.pobDesktopListening) window.dispatchEvent(new CustomEvent("pob:desktop-import", { detail: %s }));
else (window.pobDesktopPending ??= []).push(%s);`, b, b))
}

func (d *desktop) toast(message string) {
	d.dispatch("pob:desktop-toast", message)
}

func (d *desktop) dispatch(event, detail string) {
	b, _ := json.Marshal(detail)
	d.window.ExecJS(fmt.Sprintf("window.dispatchEvent(new CustomEvent(%q, { detail: %s }))", event, b))
}

func pageHandler() (http.Handler, error) {
	if dev := os.Getenv("GOPOB_DESKTOP_DEVSERVER"); dev != "" {
		u, err := url.Parse(dev)
		if err != nil {
			return nil, fmt.Errorf("GOPOB_DESKTOP_DEVSERVER: %w", err)
		}
		return httputil.NewSingleHostReverseProxy(u), nil //nolint:gosec // the developer's own vite dev server
	}
	assets, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, fmt.Errorf("embedded dist: %w", err)
	}
	built, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return nil, fmt.Errorf("desktop/dist has no index.html (build it with `task build-desktop`): %w", err)
	}
	index, err := host.DesktopIndex(built)
	if err != nil {
		return nil, fmt.Errorf("desktop index: %w", err)
	}
	return host.SPA(assets, index), nil
}
