package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type FetchRecord struct {
	URL   string     `json:"url"`
	Host  string     `json:"host"`
	Class fetchClass `json:"class"`
	HTTP  int        `json:"http"`
	Via   string     `json:"via"`
	Bytes int        `json:"bytes"`
	Blob  string     `json:"blob,omitempty"`
	Note  string     `json:"note,omitempty"`
	At    string     `json:"at"`
}

func cmdFetch(args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory for all stage artifacts")
	rate := fs.Duration("rate", 900*time.Millisecond, "minimum interval between requests to one host")
	proxyRate := fs.Duration("proxy-rate", 1500*time.Millisecond, "minimum interval for the read-proxy host")
	useProxy := fs.Bool("proxy", true, "fall back to the r.jina.ai read proxy when a host blocks us")
	useRaw := fs.Bool("raw-endpoints", false, "request /raw instead of the page — ⚠️ robots.txt DISALLOWS /raw on both hosts")
	limit := fs.Int("limit", 0, "stop after N fetches (0 = all); use for a pilot")
	retryBlocked := fs.Bool("retry-blocked", true, "re-attempt URLs previously recorded blocked/transport")
	workers := fs.Int("workers", 2, "concurrent hosts (per-host serialisation is enforced regardless)")
	verbose := fs.Bool("v", false, "log every request")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("cmdFetch: %w", err)
	}

	idx, err := loadIndex(*work)
	if err != nil {
		return err
	}
	if len(idx.byID) == 0 {
		return fmt.Errorf("link index %s is empty or missing — run `discover` first", idx.path)
	}

	done, err := loadFetchLog(*work)
	if err != nil {
		return err
	}

	rawDir := filepath.Join(*work, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return fmt.Errorf("cmdFetch: %w", err)
	}

	todo := make([]Link, 0, len(idx.byID))
	skipped := map[fetchClass]int{}
	for _, l := range idx.links() {
		if rec, ok := done[l.URL]; ok {
			if rec.Class.terminal() || (!*retryBlocked && rec.Class != fetchOK) {
				skipped[rec.Class]++
				continue
			}
		}
		todo = append(todo, l)
	}
	if *limit > 0 && len(todo) > *limit {
		todo = todo[:*limit]
	}

	banner("fetch: %d indexed, %d already terminal, %d to attempt", len(idx.byID), len(idx.byID)-len(todo)-0, len(todo))
	if len(skipped) > 0 {
		fmt.Print("  resumed past: ")
		for _, k := range sortedClasses(skipped) {
			fmt.Printf("%s=%d ", k, skipped[k])
		}
		fmt.Println()
	}
	if len(todo) == 0 {
		fmt.Println("nothing to do")
		return nil
	}

	lim := newHostLimiter(*rate, map[string]time.Duration{"r.jina.ai": *proxyRate})
	c := newClient(lim, browserUA, *verbose)

	logFile, err := os.OpenFile(filepath.Join(*work, "fetch.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("cmdFetch: %w", err)
	}
	defer logFile.Close()

	var (
		mu         sync.Mutex
		enc        = json.NewEncoder(logFile)
		counts     = map[fetchClass]int{}
		viaCnt     = map[string]int{}
		t0         = time.Now()
		ctx        = context.Background()
		jobs       = make(chan Link)
		wg         sync.WaitGroup
		nDone      int
		viaProxyOK int
	)

	for i := 0; i < max(1, *workers); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for l := range jobs {
				rec := fetchOne(ctx, c, rawDir, l, *useProxy, *useRaw)
				mu.Lock()
				counts[rec.Class]++
				viaCnt[rec.Via]++
				if rec.Class == fetchOK && rec.Via == "proxy" {
					viaProxyOK++
				}
				nDone++
				if err := enc.Encode(rec); err != nil {
					fmt.Fprintf(os.Stderr, "  warn: writing fetch log: %v\n", err)
				}
				if nDone%50 == 0 || nDone == len(todo) {
					fmt.Printf("  %5d/%d  ok=%d blocked=%d notfound=%d notcode=%d transport=%d  (proxy-recovered %d)  %s\n",
						nDone, len(todo), counts[fetchOK], counts[fetchBlocked], counts[fetchNotFound],
						counts[fetchNotCode], counts[fetchTransport], viaProxyOK, fmtDur(time.Since(t0)))
				}
				mu.Unlock()
			}
		}()
	}
	for _, l := range todo {
		jobs <- l
	}
	close(jobs)
	wg.Wait()

	fmt.Printf("\nattempted       %d in %s\n", nDone, fmtDur(time.Since(t0)))
	for _, k := range sortedClasses(counts) {
		fmt.Printf("  %-12s %6d\n", k, counts[k])
	}
	fmt.Printf("proxy recovered %d of the %d that the direct path could not reach\n",
		viaProxyOK, viaProxyOK+counts[fetchBlocked])

	if blocked := counts[fetchBlocked] + counts[fetchTransport]; blocked > 0 {
		pct := 100 * float64(blocked) / float64(nDone)
		fmt.Printf("\n⚠️  %d of %d (%.1f%%) ended BLOCKED or TRANSPORT — these are NOT rejections.\n", blocked, nDone, pct)
		fmt.Println("   Re-run `fetch` to retry exactly those; they stay non-terminal in the log.")
		if pct > 20 {
			fmt.Println("   Above 20%: treat this harvest as INCOMPLETE and do not validate it as final.")
		}
	}
	return nil
}

func fetchOne(ctx context.Context, c *client, rawDir string, l Link, useProxy, useRaw bool) FetchRecord {
	now := time.Now().UTC().Format(time.RFC3339)
	spec := hostSpecFor(l.Host)
	if spec == nil {
		return FetchRecord{URL: l.URL, Host: l.Host, Class: fetchUnsup, At: now,
			Note: "no host spec knows how to fetch this host"}
	}
	raw := useRaw || spec.rawAllowed
	targetURL := spec.canonical(l.ID())
	if raw {
		targetURL = spec.raw(l.ID())
	}

	attempt := func(host, u, via string, post func([]byte) []byte) (FetchRecord, bool) {
		r := c.get(ctx, host, u, !raw)
		rec := FetchRecord{URL: l.URL, Host: l.Host, Class: r.class, HTTP: r.code, Via: via,
			Bytes: len(r.body), Note: r.note, At: now}
		if r.class != fetchOK {
			return rec, false
		}
		body := r.body
		if post != nil {
			body = post(body)
		}
		if code, ok := extractEmbeddedCode(body); ok {
			body = code
		}
		if !looksLikeExportCode(body) {
			rec.Class = fetchNotCode
			rec.Note = fmt.Sprintf("%d bytes, no PoB code embedded", len(body))
			return rec, false
		}
		name := sha1hex([]byte(l.URL)) + ".txt"
		if err := os.WriteFile(filepath.Join(rawDir, name), body, 0o600); err != nil {
			rec.Class = fetchTransport
			rec.Note = "write blob: " + err.Error()
			return rec, false
		}
		rec.Blob, rec.Bytes = name, len(body)
		return rec, true
	}

	rec, ok := attempt(l.Host, targetURL, "direct", nil)
	if ok || rec.Class.terminal() || !useProxy {
		return rec
	}
	if rec.Class != fetchBlocked && rec.Class != fetchTransport {
		return rec
	}
	prec, pok := attempt("r.jina.ai", proxyURL(targetURL), "proxy", stripProxyPreamble)
	if pok {
		return prec
	}
	rec.Note = strings.TrimSpace(rec.Note + " | proxy: " + string(prec.Class) + " " + prec.Note)
	return rec
}

func (l Link) ID() string {
	i := strings.LastIndex(l.URL, "/")
	if i < 0 {
		return l.URL
	}
	return l.URL[i+1:]
}

func loadFetchLog(work string) (map[string]FetchRecord, error) {
	out := map[string]FetchRecord{}
	f, err := os.Open(filepath.Join(work, "fetch.jsonl"))
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loadFetchLog: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r FetchRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		if prev, ok := out[r.URL]; ok && prev.Class == fetchOK && r.Class != fetchOK {
			continue
		}
		out[r.URL] = r
	}
	return out, wrapScan(sc.Err())
}

func sortedClasses(m map[fetchClass]int) []fetchClass {
	out := make([]fetchClass, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return m[out[i]] > m[out[j]] })
	return out
}
