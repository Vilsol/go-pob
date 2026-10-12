package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	waybackHost = "web.archive.org"
	waybackBase = "https://" + waybackHost
)

type waybackSnapshot struct {
	original  string
	timestamp string
}

func parseCDXRows(body []byte) ([]waybackSnapshot, error) {
	text := string(body)
	if head := strings.ToLower(strings.TrimSpace(text)); strings.HasPrefix(head, "<") ||
		strings.Contains(head[:min(len(head), 400)], "service unavailable") {
		return nil, fmt.Errorf("wayback: the CDX endpoint answered with a page, not an index: %s",
			strings.TrimSpace(text[:min(len(text), 200)]))
	}

	var out []waybackSnapshot
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if len(fields[1]) != 14 {
			continue
		}
		out = append(out, waybackSnapshot{original: fields[0], timestamp: fields[1]})
	}
	return out, nil
}

func waybackReplayURL(s waybackSnapshot) string {
	return waybackBase + "/web/" + s.timestamp + "/" + s.original
}

func snapshotTime(ts string) (time.Time, error) {
	t, err := time.ParseInLocation("20060102150405", ts, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("wayback: unparseable snapshot timestamp %q: %w", ts, err)
	}
	return t, nil
}

func waybackCheckpointKey(s waybackSnapshot) string {
	return "wayback/" + s.timestamp + "/" + s.original
}

func discoverWayback(ctx context.Context, c *client, idx *linkIndex, st *discoverStats, ck *checkpoints,
	pattern string, start, end time.Time, verbose bool,
) error {
	if pattern == "" {
		return fmt.Errorf("wayback: -wb-url is required (e.g. 'pobarchives.com/build/*') — " +
			"this arm reads an archive by URL pattern and has no default worth guessing")
	}

	cdx := fmt.Sprintf("%s/cdx/search/cdx?url=%s&output=text&fl=original,timestamp&collapse=urlkey&from=%s&to=%s",
		waybackBase, url.QueryEscape(pattern), start.Format("20060102"), end.Format("20060102"))
	body, err := getWithRetry(ctx, c, waybackHost, cdx, 5, verbose, false)
	st.requests++
	if err != nil {
		return fmt.Errorf("wayback: querying the CDX index for %s: %w", pattern, err)
	}
	snaps, err := parseCDXRows(body)
	if err != nil {
		return err
	}
	fmt.Printf("  wayback       %d snapshots of %s between %s and %s\n",
		len(snaps), pattern, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if len(snaps) == 0 {
		fmt.Println("  wayback       nothing archived in this window — that is an answer about the ARCHIVE, not about the host")
		return nil
	}

	t0 := time.Now()
	before := len(idx.byID)
	empty, resumed := 0, 0
	for i, s := range snaps {
		if ck.complete(waybackCheckpointKey(s)) {
			resumed++
			continue
		}
		replay := waybackReplayURL(s)
		page, err := getWithRetry(ctx, c, waybackHost, replay, 6, verbose, true)
		st.requests++
		if err != nil {
			return fmt.Errorf("wayback snapshot %s (%d of %d): %w", replay, i+1, len(snaps), err)
		}

		at, err := snapshotTime(s.timestamp)
		if err != nil {
			return err
		}

		n := len(idx.byID)
		st.harvest(idx, string(page), "wayback:"+hostOf(s.original), replay, at.Unix())
		if len(idx.byID) == n {
			empty++
		}
		ck.done(waybackCheckpointKey(s))
		st.flush(false)

		if (i+1)%25 == 0 {
			fmt.Printf("    %4d/%d  %d links  %d rate-sheds  %s\n",
				i+1, len(snaps), len(idx.byID)-before, c.shedCount(waybackHost), fmtDur(time.Since(t0)))
		}
	}
	fmt.Printf("  wayback       %d new links from %d snapshots (%d carried nothing new, %d skipped as already crawled)\n",
		len(idx.byID)-before, len(snaps)-resumed, empty, resumed)
	fmt.Printf("  wayback       %d rate-sheds from %s in %s\n",
		c.shedCount(waybackHost), waybackHost, fmtDur(time.Since(t0)))
	return nil
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "unknown"
	}
	return strings.TrimPrefix(u.Host, "www.")
}
