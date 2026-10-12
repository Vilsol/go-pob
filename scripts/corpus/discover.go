package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var errMissingWindow = errors.New("-from and -to are required (YYYY-MM-DD, UTC)")

func cmdDiscover(args []string) error {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory for all stage artifacts")
	source := fs.String("source", "reddit", "source to crawl: reddit | forum | poebuilds | pobarchives | wayback")
	subs := fs.String("subs", "PathOfExileBuilds", "reddit: comma-separated subreddits")
	kinds := fs.String("kinds", "posts,comments", "reddit: which arctic-shift endpoints to sweep")
	forums := fs.String("forums", "40,marauder,24,436,303,41,22", "forum: comma-separated class build-forum ids")
	titleRe := fs.String("title-match", `\[\s*3\.19`, "forum: only descend into threads whose title matches")
	maxPages := fs.Int("max-listing-pages", 400, "forum: listing pages per forum before giving up")
	pbLeagueFlag := fs.String("pb-league", "", "poebuilds: league slug or version (e.g. mirage or 3.28); empty = the site's current league")
	pbRate := fs.Duration("pb-rate", 2*time.Second, "poebuilds: minimum interval between requests")
	pbAuthors := fs.Bool("pb-authors", false, "poebuilds: sweep the sitemap's ~267 AUTHOR pages instead of the 21 ascendancy pages (no league filter)")
	paTree := fs.String("pa-tree", "", "pobarchives: tree version to filter on (e.g. 3.29); REQUIRED for that source")
	paRate := fs.Duration("pa-rate", 1500*time.Millisecond, "pobarchives: minimum interval between requests")
	paCDP := fs.Bool("pa-cdp", false, "pobarchives: render pages in a real Chromium over CDP (REQUIRED while Cloudflare challenges this host)")
	cdpBin := fs.String("cdp-chromium", "chromium", "path to the Chromium executable used by -pa-cdp")
	cdpProfile := fs.String("cdp-profile", "", "persistent user-data-dir for -pa-cdp; empty = a temp profile discarded on exit")
	cdpHeadless := fs.Bool("cdp-headless", false, "run the CDP browser headless — MEASURED NOT TO CLEAR THE CHALLENGE; for A/B only")
	wbURL := fs.String("wb-url", "", "wayback: CDX url pattern to enumerate (e.g. 'pobarchives.com/build/*'); REQUIRED for that source")
	wbRate := fs.Duration("wb-rate", 5*time.Second, "wayback: minimum interval between requests to web.archive.org — a replayed page is ~200 KB and the host sheds load with a 503 well before it refuses")
	from := fs.String("from", "", "window start, YYYY-MM-DD (UTC, inclusive)")
	to := fs.String("to", "", "window end, YYYY-MM-DD (UTC, exclusive)")
	rate := fs.Duration("rate", 350*time.Millisecond, "minimum interval between requests to one host")
	verbose := fs.Bool("v", false, "log every request")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("cmdDiscover: %w", err)
	}

	var start, end time.Time
	if *source != "poebuilds" && *source != "pobarchives" {
		var err error
		if start, end, err = parseWindow(*from, *to); err != nil {
			return err
		}
	}

	idx, err := loadIndex(*work)
	if err != nil {
		return err
	}
	before := len(idx.byID)

	ck, err := loadCheckpoints(*work)
	if err != nil {
		return err
	}

	lim := newHostLimiter(*rate, map[string]time.Duration{
		poebuildsHost:   *pbRate,
		pobarchivesHost: *paRate,
		waybackHost:     *wbRate,
	})
	c := newClient(lim, browserUA, *verbose)
	ctx := context.Background()

	switch *source {
	case "poebuilds":
		league := *pbLeagueFlag
		if league == "" {
			league = "(site default)"
		}
		if *pbAuthors {
			league = "ALL (author pages carry no league filter)"
		}
		banner("discover: source=%s league=%s", *source, league)
	case "pobarchives":
		banner("discover: source=%s tree=%s", *source, *paTree)
	case "wayback":
		banner("discover: source=%s url=%s window=%s .. %s", *source, *wbURL,
			start.Format("2006-01-02"), end.Format("2006-01-02"))
	default:
		banner("discover: source=%s window=%s .. %s", *source, start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	t0 := time.Now()
	stats := discoverStats{ck: ck, idx: idx, lastSave: time.Now()}
	switch *source {
	case "reddit":
		err = discoverReddit(ctx, c, idx, &stats, strings.Split(*subs, ","), strings.Split(*kinds, ","), start, end, *verbose)
	case "forum":
		err = discoverForum(ctx, c, idx, &stats, strings.Split(*forums, ","), *titleRe, *maxPages, start, end, *verbose)
	case "poebuilds":
		if *pbAuthors {
			err = discoverPoebuildsAuthors(ctx, c, idx, &stats, *verbose)
		} else {
			err = discoverPoebuilds(ctx, c, idx, &stats, *pbLeagueFlag, *verbose)
		}
	case "pobarchives":
		var br *cdpBrowser
		if *paCDP {
			br, err = launchBrowser(ctx, cdpOptions{
				binary:   *cdpBin,
				profile:  *cdpProfile,
				headless: *cdpHeadless,
				verbose:  *verbose,
			})
			if err != nil {
				return err
			}
			fmt.Printf("  transport     Chromium over CDP (%s, headless=%v)\n", *cdpBin, *cdpHeadless)
			defer func() {
				if cerr := br.close(); cerr != nil {
					fmt.Fprintf(os.Stderr, "  warn: closing browser: %v\n", cerr)
				}
			}()
		}
		err = discoverPobarchives(ctx, c, idx, &stats, ck, *paTree, *verbose, br)
	case "wayback":
		err = discoverWayback(ctx, c, idx, &stats, ck, *wbURL, start, end, *verbose)
	default:
		return fmt.Errorf("unknown -source %q (want reddit, forum, poebuilds, pobarchives or wayback)", *source)
	}
	if serr := idx.save(); serr != nil {
		return fmt.Errorf("discover: %w", errors.Join(err, serr))
	}
	if serr := ck.save(); serr != nil {
		return fmt.Errorf("discover: %w", errors.Join(err, serr))
	}
	if err != nil {
		fmt.Printf("\nCRAWL INCOMPLETE — index saved with %d links (%d new) and progress checkpointed.\n",
			len(idx.byID), len(idx.byID)-before)
		fmt.Println("Re-run the identical command to resume from the checkpoint.")
		return err
	}

	fmt.Printf("\nscanned         %d items (%d carried a link)\n", stats.items, stats.itemsWithLink)
	fmt.Printf("requests        %d in %s\n", stats.requests, fmtDur(time.Since(t0)))
	fmt.Printf("links           %d new, %d already indexed, %d total\n",
		len(idx.byID)-before, stats.dupes, len(idx.byID))
	if len(stats.badIDs) > 0 {
		fmt.Printf("regex rejects   %v  (matched a host but failed its id shape — NOT silently dropped)\n", stats.badIDs)
	}
	fmt.Println("\nby host:")
	for _, r := range countBy(idx.links(), func(l Link) string { return l.Host }) {
		fmt.Printf("  %-24s %6d\n", r.Key, r.N)
	}
	fmt.Println("by source:")
	for _, r := range countBy(idx.links(), func(l Link) string { return l.Source }) {
		fmt.Printf("  %-24s %6d\n", r.Key, r.N)
	}
	fmt.Printf("\nindex -> %s\n", idx.path)
	return nil
}

type discoverStats struct {
	items         int
	itemsWithLink int
	requests      int
	dupes         int
	badIDs        map[string]int
	ck            *checkpoints
	idx           *linkIndex
	lastSave      time.Time
}

func (s *discoverStats) flush(force bool) {
	if s.idx == nil {
		return
	}
	if !force && time.Since(s.lastSave) < 30*time.Second {
		return
	}
	s.lastSave = time.Now()
	if err := s.idx.save(); err != nil {
		fmt.Fprintf(os.Stderr, "  warn: intermediate index save failed: %v\n", err)
	}
	if err := s.ck.save(); err != nil {
		fmt.Fprintf(os.Stderr, "  warn: intermediate checkpoint save failed: %v\n", err)
	}
}

func (s *discoverStats) note(rejected map[string]int) {
	if s.badIDs == nil {
		s.badIDs = map[string]int{}
	}
	for k, v := range rejected {
		s.badIDs[k] += v
	}
}

func (s *discoverStats) harvest(idx *linkIndex, text, source, context string, postedAt int64) {
	s.items++
	found, rejected := extractLinks(text)
	s.note(rejected)
	if len(found) == 0 {
		return
	}
	s.itemsWithLink++
	for _, f := range found {
		spec := hostSpecFor(f.Host)
		posted := ""
		if postedAt != 0 {
			posted = utc(postedAt)
		}
		l := Link{
			URL:       spec.canonical(f.ID),
			Host:      f.Host,
			Source:    source,
			PostedAt:  posted,
			Context:   context,
			FirstSeen: time.Now().UTC().Format(time.RFC3339),
		}
		if !idx.add(l) {
			s.dupes++
		}
	}
}

const arcticBase = "https://arctic-shift.photon-reddit.com/api"

const browserUA = "go-pob-corpus/1.0 (+https://github.com/Vilsol/go-pob; build-corpus research; contact via repo issues) Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

type arcticResponse struct {
	Data  []json.RawMessage `json:"data"`
	Error string            `json:"error"`
}

type arcticItem struct {
	ID        string  `json:"id"`
	CreatedAt float64 `json:"created_utc"`
	Title     string  `json:"title"`
	Selftext  string  `json:"selftext"`
	Body      string  `json:"body"`
	URL       string  `json:"url"`
	LinkID    string  `json:"link_id"`
}

func permalinkFor(sub string, it arcticItem) string {
	if it.LinkID != "" {
		return fmt.Sprintf("https://www.reddit.com/r/%s/comments/%s/_/%s",
			sub, strings.TrimPrefix(it.LinkID, "t3_"), it.ID)
	}
	if strings.HasPrefix(it.URL, "https://www.reddit.com/") {
		return it.URL
	}
	return fmt.Sprintf("https://www.reddit.com/r/%s/comments/%s/", sub, it.ID)
}

func discoverReddit(ctx context.Context, c *client, idx *linkIndex, st *discoverStats, subs, kinds []string, start, end time.Time, verbose bool) error {
	for _, sub := range subs {
		sub = strings.TrimSpace(sub)
		if sub == "" {
			continue
		}
		for _, kind := range kinds {
			kind = strings.TrimSpace(kind)
			if kind == "" {
				continue
			}
			endpoint, fields := "", ""
			switch kind {
			case "posts":
				endpoint, fields = "posts", "id,created_utc,title,selftext,url"
			case "comments":
				endpoint, fields = "comments", "id,created_utc,body,link_id"
			default:
				return fmt.Errorf("unknown reddit kind %q", kind)
			}

			n, err := sweepArctic(ctx, c, idx, st, sub, endpoint, fields, start, end, verbose)
			fmt.Printf("  r/%-20s %-8s %6d items\n", sub, kind, n)
			if err != nil {
				return err
			}
			st.ck.done(arcticCheckpointKey(sub, endpoint, start, end))
		}
	}
	return nil
}

func sweepArctic(ctx context.Context, c *client, idx *linkIndex, st *discoverStats, sub, endpoint, fields string, start, end time.Time, verbose bool) (int, error) {
	key := arcticCheckpointKey(sub, endpoint, start, end)
	if st.ck.complete(key) {
		fmt.Printf("  r/%-20s %-8s already complete for this window (checkpoint)\n", sub, endpoint)
		return 0, nil
	}
	cursor := st.ck.resume(key, start)
	if cursor > start.Unix() {
		fmt.Printf("  r/%-20s %-8s resuming at %s\n", sub, endpoint, utc(cursor))
	}
	endUnix := end.Unix()
	seen := map[string]bool{}
	items := 0

	for cursor < endUnix {
		q := url.Values{}
		q.Set("subreddit", sub)
		q.Set("after", fmt.Sprint(cursor))
		q.Set("before", fmt.Sprint(endUnix))
		q.Set("limit", "100")
		q.Set("sort", "asc")
		q.Set("fields", fields)
		u := fmt.Sprintf("%s/%s/search?%s", arcticBase, endpoint, q.Encode())

		body, err := getWithRetry(ctx, c, "arctic-shift.photon-reddit.com", u, 5, verbose, true)
		st.requests++
		if err != nil {
			return items, fmt.Errorf("r/%s %s at cursor %s (%s): %w",
				sub, endpoint, utc(cursor), fmtProgress(start, end, cursor), err)
		}

		var resp arcticResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return items, fmt.Errorf("r/%s %s: decoding response at cursor %s: %w", sub, endpoint, utc(cursor), err)
		}
		if resp.Error != "" {
			return items, fmt.Errorf("r/%s %s: api error at cursor %s: %s", sub, endpoint, utc(cursor), resp.Error)
		}
		if len(resp.Data) == 0 {
			return items, nil
		}

		var last float64
		for _, raw := range resp.Data {
			var it arcticItem
			if err := json.Unmarshal(raw, &it); err != nil {
				continue
			}
			last = it.CreatedAt
			if seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			items++

			text := it.Title + "\n" + it.Selftext + "\n" + it.Body + "\n" + it.URL
			st.harvest(idx, text, "r/"+sub, permalinkFor(sub, it), int64(it.CreatedAt))
		}

		next := int64(last)
		if next <= cursor {
			fmt.Fprintf(os.Stderr, "  warn: >100 %s in one second at %s in r/%s — stepping over, some may be skipped\n",
				endpoint, utc(cursor), sub)
			next = cursor + 1
		}
		cursor = next
		st.ck.mark(key, cursor)
		st.flush(false)

		if verbose {
			fmt.Printf("    r/%s %s %s (%d items)\n", sub, endpoint, utc(cursor), items)
		}
	}
	return items, nil
}

func fmtProgress(start, end time.Time, cursor int64) string {
	total := end.Unix() - start.Unix()
	if total <= 0 {
		return "?"
	}
	return fmt.Sprintf("%.0f%% through the window", 100*float64(cursor-start.Unix())/float64(total))
}

var (
	reThreadLink  = regexp.MustCompile(`href="/forum/view-thread/(\d+)"[^>]*>\s*([^<]*)`)
	reLastPageNum = regexp.MustCompile(`/forum/view-thread/\d+/page/(\d+)`)
)

func discoverForum(ctx context.Context, c *client, idx *linkIndex, st *discoverStats, forums []string, titleRe string, maxPages int, start, end time.Time, verbose bool) error {
	title, err := regexp.Compile("(?i)" + titleRe)
	if err != nil {
		return fmt.Errorf("-title-match: %w", err)
	}

	threads := map[string]string{}
	seenThread := map[string]bool{}
	for _, f := range forums {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		matched := 0
		for page := 1; page <= maxPages; page++ {
			u := fmt.Sprintf("https://www.pathofexile.com/forum/view-forum/%s/page/%d", f, page)
			body, err := getWithRetry(ctx, c, "www.pathofexile.com", u, 4, verbose, true)
			st.requests++
			if err != nil {
				return fmt.Errorf("forum %s listing page %d: %w", f, page, err)
			}
			ms := reThreadLink.FindAllStringSubmatch(string(body), -1)
			if len(ms) == 0 {
				break
			}
			fresh := 0
			for _, m := range ms {
				id, tt := m[1], strings.TrimSpace(m[2])
				if seenThread[id] {
					continue
				}
				seenThread[id] = true
				fresh++
				if title.MatchString(tt) {
					threads[id] = tt
					matched++
				}
			}
			if fresh == 0 {
				break
			}
		}
		fmt.Printf("  forum %-12s %4d threads matching %s\n", f, matched, titleRe)
	}

	for id, tt := range threads {
		pages := 1
		for page := 1; page <= pages && page <= 60; page++ {
			u := fmt.Sprintf("https://www.pathofexile.com/forum/view-thread/%s/page/%d", id, page)
			body, err := getWithRetry(ctx, c, "www.pathofexile.com", u, 4, verbose, true)
			st.requests++
			if err != nil {
				return fmt.Errorf("thread %s page %d: %w", id, page, err)
			}
			if page == 1 {
				for _, m := range reLastPageNum.FindAllStringSubmatch(string(body), -1) {
					var n int
					if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil && n > pages {
						pages = n
					}
				}
			}
			st.harvest(idx, string(body), "poe-forum", "https://www.pathofexile.com/forum/view-thread/"+id, start.Unix())
			st.flush(false)
			_ = tt
		}
	}
	fmt.Printf("  descended into %d threads\n", len(threads))
	return nil
}

func getWithRetry(ctx context.Context, c *client, host, u string, attempts int, verbose, htmlExpected bool) ([]byte, error) {
	var lastNote string
	for i := 1; i <= attempts; i++ {
		r := c.get(ctx, host, u, htmlExpected)
		if verbose {
			fmt.Printf("      GET %s -> %s %d\n", u, r.class, r.code)
		}
		if r.class == fetchOK {
			return r.body, nil
		}
		if r.class.terminal() {
			return nil, fmt.Errorf("%s (HTTP %d): %s", r.class, r.code, r.note)
		}
		lastNote = fmt.Sprintf("%s (HTTP %d) %s", r.class, r.code, r.note)
		time.Sleep(time.Duration(i) * 2 * time.Second)
	}
	return nil, fmt.Errorf("%d attempts failed, last: %s", attempts, lastNote)
}

func parseWindow(from, to string) (time.Time, time.Time, error) {
	if from == "" || to == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("parseWindow: %w", errMissingWindow)
	}
	s, err := time.ParseInLocation("2006-01-02", from, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("-from: %w", err)
	}
	e, err := time.ParseInLocation("2006-01-02", to, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("-to: %w", err)
	}
	if !e.After(s) {
		return time.Time{}, time.Time{}, fmt.Errorf("-to (%s) must be after -from (%s)", to, from)
	}
	return s, e, nil
}

func defaultWork() string {
	if w := os.Getenv("CORPUS_WORK"); w != "" {
		return w
	}
	return "corpus-work"
}
