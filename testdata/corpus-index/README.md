# Per-league candidate link indexes — the durable half of a corpus harvest

One `index.jsonl.gz` per league. Each line is **one candidate PoB link ever seen**, winner
or not, with where it was found, whether we could reach it, and what we decided about it.

```
testdata/corpus-index/<league>/index.jsonl.gz
```

Produced and consumed by `scripts/corpus`:

```shell
mise run task -- corpus-archive  LEAGUE=3.19 WORK=~/go-pob-corpus-319/full   # write it
mise run task -- corpus-seed     LEAGUE=3.19 WORK=~/go-pob-corpus-319/redo   # re-crawl nothing
mise run task -- corpus-linkrot  LEAGUE=3.18                                 # how much still resolves
```

## ⚠️ Why this exists — the 3.18 harvest threw its own crawl away

That run discovered ~6100 links and fetched 3496 codes. **All of that is gone.** Only the
2027 winners' URLs survive, embedded in `corpus-3.18/MANIFEST.md` prose. The ~1400 rejected
candidates, and the crawl that found them, are unrecoverable — so a re-run starts from zero.

**Re-fetching is cheap. Re-crawling is not**, and it gets more expensive every year: the
archive that finds the links is a third party (pushshift died and was replaced by
arctic-shift), and the pastes it points at are deleted continuously.

Keeping the REJECTS is the part that is easy to skip and expensive to lose. A list of
winners cannot distinguish *"this source is exhausted"* from *"this source was never
tried"*; a list of every candidate with its verdict can.

## Fields

| field | |
|---|---|
| `url` `host` `source` `posted_at` `context` | discovery: the canonical paste URL and the post/comment that carried it |
| `first_seen` | when **we** indexed it — the denominator for link rot, unlike `posted_at`. Empty means "indexed before the tool recorded it" |
| `last_fetch_status` `last_fetch_at` `http` `via` | transport, from the closed class set in `scripts/corpus/http.go`. **`blocked` is not a rejection** |
| `verdict` `detail` | `accepted`, or a rejection reason from the closed set in `scripts/corpus/pob.go` |
| `file` `sha1` | the corpus filename and the SHA-1 of the decompressed XML, when accepted |

The raw export codes are **not** archived — only their addresses. Refetching 2000 links is
~25 minutes; re-crawling the league that found them is hours and degrades with age.

## What each league's file actually contains

| league | rows | note |
|---|---|---|
| **3.18** | 2027 | ⚠️ **RECONSTRUCTED from `corpus-3.18/MANIFEST.md`, winners only.** The rejects are gone for good. `source` counts reproduce the manifest's stated provenance exactly (r/PathOfExileBuilds 1819, poe-forum 112, r/pathofexile 96), which is what makes the reconstruction trustworthy. `first_seen` is empty throughout: the original crawl did not record it |
| **3.19** | 1047 | the **2-week pilot only** (2022-08-19..09-02), not a full-window harvest. Complete for that window: 919 accepted and 128 rejected, each with its reason. 3.19 is NOT a harvest target — `go-pob-data` has no 3.19 |
| **3.20** | 7968 | the **full-window harvest** (2022-12-09..2023-04-07), reddit both subs + all 7 class forums. 6105 accepted -> `testdata/corpus-3.20/`, 1863 rejected with reasons, 0 ever blocked |

Neither file is a gate and nothing asserts against them. They are inputs.
