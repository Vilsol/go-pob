package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	gggHost       = "www.pathofexile.com"
	ladderKey     = "www.pathofexile.com#ladder"
	charKey       = "www.pathofexile.com#character"
	gggBase       = "https://" + gggHost
	ladderPageMax = 200
)

type ladderEntry struct {
	Rank    int  `json:"rank"`
	Dead    bool `json:"dead"`
	Public  bool `json:"public"`
	Account struct {
		Name string `json:"name"`
	} `json:"account"`
	Character struct {
		Name  string `json:"name"`
		Level int    `json:"level"`
		Class string `json:"class"`
	} `json:"character"`
}

type ladderPage struct {
	Total   int           `json:"total"`
	Entries []ladderEntry `json:"entries"`
}

func intervalFromPolicy(header string) time.Duration {
	worst := time.Duration(0)
	for _, rule := range strings.Split(header, ",") {
		parts := strings.Split(strings.TrimSpace(rule), ":")
		if len(parts) < 2 {
			continue
		}
		hits, err1 := strconv.Atoi(parts[0])
		period, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || hits <= 0 || period <= 0 {
			continue
		}
		iv := time.Duration(period) * time.Second / time.Duration(hits)
		if iv > worst {
			worst = iv
		}
	}
	return worst
}

const safetyMargin = 1.1

func characterID(account, character string) string {
	repl := strings.NewReplacer("#", "_", "/", "_", " ", "_", "\\", "_", ":", "_")
	return repl.Replace(account) + "." + repl.Replace(character)
}

func cmdLadder(args []string) error {
	fs := flag.NewFlagSet("ladder", flag.ExitOnError)
	work := fs.String("work", defaultWork(), "working directory")
	ladders := fs.String("ladders", "", "comma-separated ladder names, e.g. 'Allflame,Hardcore Allflame'")
	maxChars := fs.Int("max", 0, "stop after this many characters fetched this run (0 = no limit)")
	minLevel := fs.Int("min-level", 68, "skip characters below this level — a low-level character has too few items to pass validate")
	verbose := fs.Bool("v", false, "log every request")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("cmdLadder: %w", err)
	}
	if strings.TrimSpace(*ladders) == "" {
		return fmt.Errorf("ladder: -ladders is required (e.g. -ladders 'Allflame,Hardcore Allflame') — " +
			"there is no sensible default and guessing one silently harvests the wrong league")
	}

	charDir := filepath.Join(*work, "characters")
	if err := os.MkdirAll(charDir, 0o755); err != nil {
		return fmt.Errorf("ladder: creating %s: %w", charDir, err)
	}
	ck, err := loadCheckpoints(*work)
	if err != nil {
		return err
	}

	lim := newHostLimiter(time.Second, map[string]time.Duration{ladderKey: 2 * time.Second, charKey: 44 * time.Second})
	c := newClient(lim, browserUA, *verbose)
	ctx := context.Background()

	banner("ladder: %s", *ladders)
	t0 := time.Now()
	fetched, skipped, private, failed := 0, 0, 0, 0

	for _, league := range strings.Split(*ladders, ",") {
		league = strings.TrimSpace(league)
		if league == "" {
			continue
		}
		for offset := 0; ; offset += ladderPageMax {
			if *maxChars > 0 && fetched >= *maxChars {
				break
			}
			page, err := fetchLadderPage(ctx, c, league, offset, *verbose)
			if err != nil {
				return fmt.Errorf("ladder %s at offset %d: %w", league, offset, err)
			}
			if len(page.Entries) == 0 {
				break
			}
			fmt.Printf("  %-22s offset %5d of %d  (%d entries)\n", league, offset, page.Total, len(page.Entries))

			for _, e := range page.Entries {
				if *maxChars > 0 && fetched >= *maxChars {
					break
				}
				switch {
				case !e.Public:
					private++
					continue
				case e.Character.Level < *minLevel:
					skipped++
					continue
				}
				id := characterID(e.Account.Name, e.Character.Name)
				if ck.complete("ladder/" + id) {
					skipped++
					continue
				}
				if err := fetchCharacter(ctx, c, charDir, id, e.Account.Name, e.Character.Name, *verbose); err != nil {
					failed++
					fmt.Fprintf(os.Stderr, "  warn: %s: %v\n", id, err)
					if failed > 20 && failed > fetched {
						return fmt.Errorf("ladder: %d failures against %d successes — refusing to continue", failed, fetched)
					}
					continue
				}
				ck.done("ladder/" + id)
				fetched++
				if err := ck.save(); err != nil {
					fmt.Fprintf(os.Stderr, "  warn: checkpoint save: %v\n", err)
				}
				if fetched%10 == 0 {
					fmt.Printf("    %5d fetched  %d skipped  %d private  %d failed  %d rate-sheds  %s\n",
						fetched, skipped, private, failed, c.shedCount(charKey), fmtDur(time.Since(t0)))
				}
			}
			if offset+ladderPageMax >= page.Total {
				break
			}
		}
	}

	if serr := ck.save(); serr != nil {
		return fmt.Errorf("ladder: %w", serr)
	}
	fmt.Printf("\nfetched      %d characters in %s\n", fetched, fmtDur(time.Since(t0)))
	fmt.Printf("skipped      %d (already held, or below -min-level)\n", skipped)
	fmt.Printf("private      %d (flagged not-public by the ladder — no request spent)\n", private)
	fmt.Printf("failed       %d\n", failed)
	fmt.Printf("rate-sheds   %d\n", c.shedCount(charKey))
	fmt.Printf("\ncharacters -> %s\nnext: mise run task -- import-characters IN=%s OUT=<builds dir>\n", charDir, charDir)
	return nil
}

func fetchLadderPage(ctx context.Context, c *client, league string, offset int, verbose bool) (*ladderPage, error) {
	u := fmt.Sprintf("%s/api/ladders/%s?limit=%d&offset=%d&realm=pc",
		gggBase, url.PathEscape(league), ladderPageMax, offset)
	body, err := getWithRetry(ctx, c, ladderKey, u, 5, verbose, false)
	if err != nil {
		return nil, err
	}
	var page ladderPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("decoding ladder page: %w", err)
	}
	return &page, nil
}

func fetchCharacter(ctx context.Context, c *client, dir, id, account, character string, verbose bool) error {
	q := "accountName=" + url.QueryEscape(account) + "&character=" + url.QueryEscape(character) + "&realm=pc"

	items, err := getWithRetry(ctx, c, charKey, gggBase+"/character-window/get-items?"+q, 4, verbose, false)
	if err != nil {
		return fmt.Errorf("get-items: %w", err)
	}
	passives, err := getWithRetry(ctx, c, charKey, gggBase+"/character-window/get-passive-skills?"+q, 4, verbose, false)
	if err != nil {
		return fmt.Errorf("get-passive-skills: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, id+".items.json"), items, 0o644); err != nil { //nolint:gosec // corpus data, not secrets
		return fmt.Errorf("writing items: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".passives.json"), passives, 0o644); err != nil { //nolint:gosec // corpus data, not secrets
		return fmt.Errorf("writing passives: %w", err)
	}
	return nil
}
