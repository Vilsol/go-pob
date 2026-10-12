package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var errCDPClosed = errors.New("cdp: browser closed the connection")

var errNoDebuggerURL = errors.New("cdp: /json/version carried no webSocketDebuggerUrl")

const maxWSFrame = 64 << 20

type wsConn struct {
	conn net.Conn
	br   *bufio.Reader
	wmu  sync.Mutex
}

func wsDial(ctx context.Context, rawURL string) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("cdp: parsing %s: %w", rawURL, err)
	}
	addr := u.Host
	if u.Port() == "" {
		addr = net.JoinHostPort(u.Hostname(), "80")
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cdp: dialing %s: %w", addr, err)
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("cdp: websocket nonce: %w", err)
	}
	handshake := "GET " + u.RequestURI() + " HTTP/1.1\r\n" +
		"Host: " + u.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + base64.StdEncoding.EncodeToString(nonce) + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := conn.Write([]byte(handshake)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("cdp: websocket handshake: %w", err)
	}

	br := bufio.NewReaderSize(conn, 64<<10)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("cdp: reading handshake response: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		return nil, fmt.Errorf("cdp: websocket upgrade refused with HTTP %d", resp.StatusCode)
	}
	return &wsConn{conn: conn, br: br}, nil
}

func (w *wsConn) write(opcode byte, payload []byte) error {
	w.wmu.Lock()
	defer w.wmu.Unlock()

	n := len(payload)
	hdr := []byte{0x80 | opcode}
	switch {
	case n < 126:
		hdr = append(hdr, 0x80|byte(n))
	case n <= 0xFFFF:
		hdr = append(hdr, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(hdr[2:], uint16(n)) //nolint:gosec // bounded by the case above
	default:
		hdr = append(hdr, 0x80|127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(hdr[2:], uint64(n)) //nolint:gosec // len is never negative
	}

	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return fmt.Errorf("cdp: frame mask: %w", err)
	}
	frame := append(hdr, mask[:]...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}
	if _, err := w.conn.Write(frame); err != nil {
		return fmt.Errorf("cdp: writing frame: %w", err)
	}
	return nil
}

func (w *wsConn) read() ([]byte, error) {
	var msg []byte
	for {
		var h [2]byte
		if _, err := io.ReadFull(w.br, h[:]); err != nil {
			return nil, fmt.Errorf("cdp: reading frame header: %w", err)
		}
		fin := h[0]&0x80 != 0
		opcode := h[0] & 0x0F
		masked := h[1]&0x80 != 0

		n := uint64(h[1] & 0x7F)
		switch n {
		case 126:
			var e [2]byte
			if _, err := io.ReadFull(w.br, e[:]); err != nil {
				return nil, fmt.Errorf("cdp: reading frame length: %w", err)
			}
			n = uint64(binary.BigEndian.Uint16(e[:]))
		case 127:
			var e [8]byte
			if _, err := io.ReadFull(w.br, e[:]); err != nil {
				return nil, fmt.Errorf("cdp: reading frame length: %w", err)
			}
			n = binary.BigEndian.Uint64(e[:])
		}
		if n > maxWSFrame {
			return nil, fmt.Errorf("cdp: frame of %d bytes exceeds the %d cap — the stream is desynchronised", n, maxWSFrame)
		}

		var mask [4]byte
		if masked {
			if _, err := io.ReadFull(w.br, mask[:]); err != nil {
				return nil, fmt.Errorf("cdp: reading frame mask: %w", err)
			}
		}
		payload := make([]byte, n)
		if _, err := io.ReadFull(w.br, payload); err != nil {
			return nil, fmt.Errorf("cdp: reading frame payload: %w", err)
		}
		if masked {
			for i := range payload {
				payload[i] ^= mask[i%4]
			}
		}

		switch opcode {
		case 0x8:
			return nil, fmt.Errorf("cdp: reading frame: %w", errCDPClosed)
		case 0x9:
			if err := w.write(0xA, payload); err != nil {
				return nil, err
			}
			continue
		case 0xA:
			continue
		}
		msg = append(msg, payload...)
		if fin {
			return msg, nil
		}
	}
}

func (w *wsConn) close() error {
	_ = w.write(0x8, nil)
	if err := w.conn.Close(); err != nil {
		return fmt.Errorf("cdp: closing connection: %w", err)
	}
	return nil
}

type cdpBrowser struct {
	cmd     *exec.Cmd
	ws      *wsConn
	profile string
	session string
	nextID  int
	verbose bool
}

type cdpOptions struct {
	binary   string
	profile  string
	headless bool
	verbose  bool
}

func launchBrowser(ctx context.Context, opt cdpOptions) (*cdpBrowser, error) {
	profile, ephemeral := opt.profile, false
	if profile == "" {
		dir, err := os.MkdirTemp("", "corpus-cdp-")
		if err != nil {
			return nil, fmt.Errorf("cdp: profile dir: %w", err)
		}
		profile, ephemeral = dir, true
	}

	args := []string{
		"--ozone-platform=x11",
		"--remote-debugging-port=0",
		"--user-data-dir=" + profile,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
		"--mute-audio",
		"--window-size=1280,900",
		"about:blank",
	}
	if opt.headless {
		args = append([]string{"--headless=new", "--disable-gpu"}, args...)
	}

	cmd := exec.CommandContext(ctx, opt.binary, args...) //nolint:gosec // the binary is an operator-supplied flag
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		if ephemeral {
			_ = os.RemoveAll(profile)
		}
		return nil, fmt.Errorf("cdp: starting %s: %w", opt.binary, err)
	}

	b := &cdpBrowser{cmd: cmd, profile: profile, verbose: opt.verbose}
	if !ephemeral {
		b.profile = ""
	}

	endpoint, err := waitForDevTools(ctx, profile, 45*time.Second)
	if err != nil {
		_ = b.close()
		return nil, err
	}
	if b.ws, err = wsDial(ctx, endpoint); err != nil {
		_ = b.close()
		return nil, err
	}

	target, err := b.send("Target.createTarget", map[string]any{"url": "about:blank"}, "")
	if err != nil {
		_ = b.close()
		return nil, err
	}
	var tgt struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(target, &tgt); err != nil {
		_ = b.close()
		return nil, fmt.Errorf("cdp: decoding createTarget: %w", err)
	}
	attached, err := b.send("Target.attachToTarget", map[string]any{"targetId": tgt.TargetID, "flatten": true}, "")
	if err != nil {
		_ = b.close()
		return nil, err
	}
	var att struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attached, &att); err != nil {
		_ = b.close()
		return nil, fmt.Errorf("cdp: decoding attachToTarget: %w", err)
	}
	b.session = att.SessionID
	return b, nil
}

func waitForDevTools(ctx context.Context, profile string, budget time.Duration) (string, error) {
	portFile := filepath.Join(profile, "DevToolsActivePort")
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("cdp: waiting for devtools: %w", ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
		raw, err := os.ReadFile(portFile) //nolint:gosec // path is composed from our own profile dir
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
		if len(lines) == 0 || lines[0] == "" {
			continue
		}
		endpoint, err := devToolsWSURL(ctx, lines[0])
		if err != nil {
			continue
		}
		return endpoint, nil
	}
	return "", fmt.Errorf("cdp: %s never appeared — Chromium did not start with remote debugging", portFile)
}

func devToolsWSURL(ctx context.Context, port string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/json/version", nil)
	if err != nil {
		return "", fmt.Errorf("cdp: /json/version request: %w", err)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("cdp: /json/version: %w", err)
	}
	defer resp.Body.Close()
	var v struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", fmt.Errorf("cdp: decoding /json/version: %w", err)
	}
	if v.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("cdp: %w", errNoDebuggerURL)
	}
	return v.WebSocketDebuggerURL, nil
}

func (b *cdpBrowser) send(method string, params map[string]any, session string) (json.RawMessage, error) {
	b.nextID++
	id := b.nextID

	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if session != "" {
		msg["sessionId"] = session
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("cdp: encoding %s: %w", method, err)
	}
	if err := b.ws.write(0x1, raw); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(60 * time.Second)
	if err := b.ws.conn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("cdp: read deadline: %w", err)
	}
	for {
		data, err := b.ws.read()
		if err != nil {
			return nil, err
		}
		var env struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		if env.ID != id {
			continue
		}
		if env.Error != nil {
			return nil, fmt.Errorf("cdp: %s: %s", method, env.Error.Message)
		}
		return env.Result, nil
	}
}

func (b *cdpBrowser) outerHTML() (string, error) {
	res, err := b.send("Runtime.evaluate", map[string]any{
		"expression":    "document.documentElement.outerHTML",
		"returnByValue": true,
	}, b.session)
	if err != nil {
		return "", err
	}
	var v struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(res, &v); err != nil {
		return "", fmt.Errorf("cdp: decoding evaluate: %w", err)
	}
	return v.Result.Value, nil
}

const cdpSettleRounds = 3

func (b *cdpBrowser) render(ctx context.Context, target string, ready func(string) bool, budget time.Duration) (string, bool, error) {
	if _, err := b.send("Page.navigate", map[string]any{"url": target}, b.session); err != nil {
		return "", false, err
	}

	deadline := time.Now().Add(budget)
	var last string
	settled := 0
	challenged := false
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", false, fmt.Errorf("cdp: rendering %s: %w", target, ctx.Err())
		case <-time.After(700 * time.Millisecond):
		}

		html, err := b.outerHTML()
		if err != nil {
			return "", false, err
		}
		if ready(html) {
			return html, true, nil
		}
		if _, blocked := looksBlocked([]byte(html), true); blocked {
			challenged, settled, last = true, 0, html
			continue
		}
		if html == last {
			settled++
			if settled >= cdpSettleRounds {
				return html, false, nil
			}
			continue
		}
		settled, last = 0, html
	}

	if challenged {
		return "", false, fmt.Errorf("cdp: %s never cleared the challenge within %s "+
			"(is Chromium running HEADFUL? headless is challenged — see the comment at the head of cdp.go)",
			target, budget)
	}
	return "", false, fmt.Errorf("cdp: %s never settled within %s", target, budget)
}

func (b *cdpBrowser) close() error {
	var errs []error
	if b.ws != nil {
		errs = append(errs, b.ws.close())
	}
	if b.cmd != nil && b.cmd.Process != nil {
		if err := b.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			errs = append(errs, fmt.Errorf("cdp: killing chromium: %w", err))
		}
		_ = b.cmd.Wait()
	}
	if b.profile != "" {
		if err := os.RemoveAll(b.profile); err != nil {
			errs = append(errs, fmt.Errorf("cdp: removing profile: %w", err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("cdp: closing browser: %w", err)
	}
	return nil
}
