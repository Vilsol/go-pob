package host

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
)

func FlatpakID() string { return flatpakID("/.flatpak-info", os.Getenv("FLATPAK_ID")) }

func flatpakID(info, id string) string {
	if _, err := os.Stat(info); err != nil {
		return ""
	}
	return id
}

type Portal struct {
	once sync.Once
	conn *dbus.Conn
	err  error
}

type portalFilter struct {
	Name     string
	Patterns []portalPattern
}

type portalPattern struct {
	Kind    uint32
	Pattern string
}

func globFilter(name, glob string) portalFilter {
	return portalFilter{Name: name, Patterns: []portalPattern{{Kind: 0, Pattern: glob}}}
}

func (p *Portal) SaveFile(suggested string) (string, error) {
	opts := map[string]dbus.Variant{"current_name": dbus.MakeVariant(suggested)}
	if ext := filepath.Ext(suggested); ext != "" {
		opts["filters"] = dbus.MakeVariant([]portalFilter{globFilter(strings.ToUpper(ext[1:])+" files", "*"+ext)})
	}
	return p.choose("SaveFile", "Save", opts)
}

func (p *Portal) OpenBuild() (string, error) {
	return p.choose("OpenFile", "Open build file", map[string]dbus.Variant{
		"filters": dbus.MakeVariant([]portalFilter{globFilter("Path of Building builds", "*.xml"), globFilter("Build codes", "*.txt")}),
	})
}

func (p *Portal) ChooseFolder(current string) (string, error) {
	opts := map[string]dbus.Variant{"directory": dbus.MakeVariant(true)}
	if current != "" {
		opts["current_folder"] = dbus.MakeVariant(append([]byte(current), 0))
	}
	return p.choose("OpenFile", "Builds folder", opts)
}

var portalTokens atomic.Uint64

func (p *Portal) choose(method, title string, opts map[string]dbus.Variant) (string, error) {
	p.once.Do(func() { p.conn, p.err = dbus.ConnectSessionBus() })
	if p.err != nil {
		return "", fmt.Errorf("file chooser portal: %w", p.err)
	}
	token := fmt.Sprintf("gopob%d", portalTokens.Add(1))
	opts["handle_token"] = dbus.MakeVariant(token)
	want := RequestPath(p.conn.Names()[0], token)
	match := []dbus.MatchOption{dbus.WithMatchObjectPath(want), dbus.WithMatchInterface("org.freedesktop.portal.Request"), dbus.WithMatchMember("Response")}
	if err := p.conn.AddMatchSignal(match...); err != nil {
		return "", fmt.Errorf("file chooser portal: %w", err)
	}
	defer func() { _ = p.conn.RemoveMatchSignal(match...) }()
	signals := make(chan *dbus.Signal, 8)
	p.conn.Signal(signals)
	defer p.conn.RemoveSignal(signals)

	var handle dbus.ObjectPath
	err := p.conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop").
		Call("org.freedesktop.portal.FileChooser."+method, 0, "", title, opts).Store(&handle)
	if err != nil {
		return "", fmt.Errorf("file chooser portal: %w", err)
	}
	for sig := range signals {
		if sig.Name == "org.freedesktop.portal.Request.Response" && (sig.Path == handle || sig.Path == want) {
			return portalResult(sig.Body)
		}
	}
	return "", fmt.Errorf("file chooser portal: %w", net.ErrClosed)
}

func RequestPath(sender, token string) dbus.ObjectPath {
	return dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + strings.ReplaceAll(strings.TrimPrefix(sender, ":"), ".", "_") + "/" + token)
}

func portalResult(body []any) (string, error) {
	if len(body) != 2 {
		return "", fmt.Errorf("file chooser portal: response has %d values", len(body))
	}
	code, _ := body[0].(uint32)
	switch code {
	case 0:
	case 1:
		return "", nil
	default:
		return "", fmt.Errorf("file chooser portal: response %d", code)
	}
	results, _ := body[1].(map[string]dbus.Variant)
	uris, _ := results["uris"].Value().([]string)
	if len(uris) == 0 {
		return "", nil
	}
	u, err := url.Parse(uris[0])
	if err != nil || u.Scheme != "file" {
		return "", fmt.Errorf("file chooser portal: not a file: %q", uris[0])
	}
	return u.Path, nil
}
