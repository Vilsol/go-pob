package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fetchClass string

const (
	fetchOK         fetchClass = "ok"
	fetchNotFound   fetchClass = "notfound"
	fetchBlocked    fetchClass = "blocked"
	fetchTransport  fetchClass = "transport"
	fetchNotCode    fetchClass = "notcode"
	fetchUnsup      fetchClass = "unsupported"
	fetchBadRequest fetchClass = "badrequest"
)

func (c fetchClass) terminal() bool {
	return c == fetchOK || c == fetchNotFound || c == fetchBadRequest
}

type httpResult struct {
	class      fetchClass
	code       int
	body       []byte
	note       string
	retryAfter time.Duration
}

type hostLimiter struct {
	mu       sync.Mutex
	interval map[string]time.Duration
	last     map[string]time.Time
	locks    map[string]*sync.Mutex
	def      time.Duration
}

func newHostLimiter(def time.Duration, per map[string]time.Duration) *hostLimiter {
	hl := &hostLimiter{
		interval: map[string]time.Duration{},
		last:     map[string]time.Time{},
		locks:    map[string]*sync.Mutex{},
		def:      def,
	}
	for k, v := range per {
		hl.interval[k] = v
	}
	return hl
}

func (hl *hostLimiter) ensureAtLeast(host string, iv time.Duration) {
	if iv <= 0 {
		return
	}
	hl.mu.Lock()
	defer hl.mu.Unlock()
	cur, ok := hl.interval[host]
	if !ok {
		cur = hl.def
	}
	if iv > cur {
		hl.interval[host] = iv
	}
}

func (hl *hostLimiter) slowDown(host string, ceiling time.Duration) time.Duration {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	iv, ok := hl.interval[host]
	if !ok {
		iv = hl.def
	}
	if iv <= 0 {
		iv = 250 * time.Millisecond
	}
	was := iv
	iv *= 2
	if iv > ceiling {
		iv = ceiling
	}
	if iv < was {
		iv = was
	}
	hl.interval[host] = iv
	return iv
}

func (hl *hostLimiter) wait(host string) func() {
	hl.mu.Lock()
	lk, ok := hl.locks[host]
	if !ok {
		lk = &sync.Mutex{}
		hl.locks[host] = lk
	}
	iv, ok := hl.interval[host]
	if !ok {
		iv = hl.def
	}
	hl.mu.Unlock()

	lk.Lock()
	hl.mu.Lock()
	prev := hl.last[host]
	hl.mu.Unlock()
	if d := time.Until(prev.Add(iv)); d > 0 {
		time.Sleep(d)
	}
	return func() {
		hl.mu.Lock()
		hl.last[host] = time.Now()
		hl.mu.Unlock()
		lk.Unlock()
	}
}

type client struct {
	http    *http.Client
	lim     *hostLimiter
	ua      string
	verbose bool

	shedMu sync.Mutex
	sheds  map[string]int
}

func (c *client) noteShed(host string) {
	c.shedMu.Lock()
	defer c.shedMu.Unlock()
	c.sheds[host]++
}

func (c *client) shedCount(host string) int {
	c.shedMu.Lock()
	defer c.shedMu.Unlock()
	return c.sheds[host]
}

func newClient(lim *hostLimiter, ua string, verbose bool) *client {
	return &client{
		http: &http.Client{
			Timeout: 45 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
		lim:     lim,
		ua:      ua,
		verbose: verbose,
		sheds:   map[string]int{},
	}
}

const rateLimitAttempts = 4

const maxHostInterval = 4 * time.Second

func isRateShed(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable
}

func (c *client) get(ctx context.Context, host, url string, htmlExpected bool) httpResult {
	var r httpResult
	for attempt := 1; ; attempt++ {
		r = c.getOnce(ctx, host, url, htmlExpected)
		if !isRateShed(r.code) || attempt >= rateLimitAttempts {
			return r
		}
		c.noteShed(host)
		iv := c.lim.slowDown(host, maxHostInterval)
		wait := r.retryAfter
		if wait < 0 {
			wait = (1 << (attempt - 1)) * time.Second
		}
		if wait < iv {
			wait = iv
		}
		if c.verbose {
			fmt.Printf("      %d from %s — waiting %v, host interval now %v\n", r.code, host, wait, iv)
		}
		select {
		case <-ctx.Done():
			return r
		case <-time.After(wait):
		}
	}
}

func (c *client) getOnce(ctx context.Context, host, url string, htmlExpected bool) httpResult {
	done := c.lim.wait(host)
	defer done()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return httpResult{class: fetchTransport, note: err.Error(), retryAfter: -1}
	}
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Accept", "text/plain, text/html;q=0.5, */*;q=0.1")

	resp, err := c.http.Do(req)
	if err != nil {
		return httpResult{class: fetchTransport, note: err.Error(), retryAfter: -1}
	}
	defer resp.Body.Close()

	if pol := resp.Header.Get("X-Rate-Limit-Ip"); pol != "" {
		if iv := intervalFromPolicy(pol); iv > 0 {
			c.lim.ensureAtLeast(host, time.Duration(float64(iv)*safetyMargin))
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return httpResult{class: fetchTransport, code: resp.StatusCode, note: "read: " + err.Error(), retryAfter: -1}
	}

	switch {
	case resp.StatusCode == http.StatusNotFound, resp.StatusCode == http.StatusGone:
		return httpResult{class: fetchNotFound, code: resp.StatusCode, retryAfter: -1}
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusTooManyRequests,
		resp.StatusCode >= 500:
		return httpResult{class: fetchBlocked, code: resp.StatusCode, note: http.StatusText(resp.StatusCode),
			retryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	case resp.StatusCode == http.StatusRequestTimeout, resp.StatusCode == http.StatusUnprocessableEntity,
		resp.StatusCode == http.StatusTooEarly:
		return httpResult{class: fetchBlocked, code: resp.StatusCode, retryAfter: -1,
			note: "load-shed: " + strings.TrimSpace(string(body[:min(len(body), 200)]))}
	case resp.StatusCode >= 400:
		return httpResult{class: fetchBadRequest, code: resp.StatusCode, retryAfter: -1,
			note: strings.TrimSpace(string(body[:min(len(body), 300)]))}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return httpResult{class: fetchBlocked, code: resp.StatusCode, note: http.StatusText(resp.StatusCode), retryAfter: -1}
	}

	if why, blocked := looksBlocked(body, htmlExpected); blocked {
		return httpResult{class: fetchBlocked, code: resp.StatusCode, body: body, note: why, retryAfter: -1}
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return httpResult{class: fetchNotCode, code: resp.StatusCode, note: "empty body", retryAfter: -1}
	}
	return httpResult{class: fetchOK, code: resp.StatusCode, body: body, retryAfter: -1}
}

var challengeMarkers = []string{
	"just a moment",
	"attention required",
	"checking your browser",
	"cf-browser-verification",
	"enable javascript and cookies",
	"__cf_chl",
	"cf-mitigated",
	"ddos protection by cloudflare",
	"please turn javascript on",
}

var htmlMarkers = []string{"<!doctype html", "<html"}

func looksBlocked(body []byte, htmlExpected bool) (string, bool) {
	head := strings.ToLower(string(body[:min(len(body), 4096)]))
	for _, m := range challengeMarkers {
		if strings.Contains(head, m) {
			return "challenge marker: " + m, true
		}
	}
	if htmlExpected {
		return "", false
	}
	for _, m := range htmlMarkers {
		if strings.Contains(head, m) {
			return "unexpected HTML on a payload endpoint: " + m, true
		}
	}
	return "", false
}

func proxyURL(raw string) string {
	return "https://r.jina.ai/" + raw
}

func stripProxyPreamble(body []byte) []byte {
	s := string(body)
	for _, marker := range []string{"Markdown Content:\n", "Content:\n"} {
		if i := strings.Index(s, marker); i >= 0 {
			s = s[i+len(marker):]
			break
		}
	}
	return []byte(strings.TrimSpace(s))
}

func fmtDur(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func parseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return -1
	}
	secs, err := strconv.Atoi(h)
	if err != nil || secs < 0 {
		return -1
	}
	return time.Duration(secs) * time.Second
}
