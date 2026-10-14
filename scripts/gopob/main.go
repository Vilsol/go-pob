package main

import (
	"fmt"
	"os"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/gateversions"
	"github.com/Vilsol/go-pob/scripts/gopob/internal/toolchain"
)

const usage = `gopob <subcommand> [flags]

  data-fields --list F | --check | project | sizes
                        which go-pob-data row fields go-pob reads (the list go-pob-data embeds as
                        fields/<version>.json), the guard against it, and a projector to try a list
  gate-versions --sh    the version table as sourceable bash, for eval "$(...)"
  gate-versions --tsv   the version table as TSV: version, oracle test, corpus test, authoritative
  kw-bits-parity        both KW_BITS tables vs mod/common.go vs the pinned PoB
  triage-drops <verdict.json> <cases.json> [top]
                        dropped mod lines split into READY (every name has a consumer)
                        and BLOCKED (a consumer must be written first)
  table-row-census <pinned ModParser.lua> <our mod_parser.go> [min_family_size]
                        per-FAMILY row counts on both sides, ranked by how many are missing
  optional-token-census <pinned ModParser.lua> <our mod_parser.go>
                        patterns we carry STRICTER than the pin's, by hardcoding an optional
  fold-progress         how much of the Python/shell tooling has a VERIFIED Go port
  citation-triage [before] [after]
                        FLAGS=<file>: each flagged //pob: range with its Go symbol beside the Lua
  citation-drift        for each //pob: citation, did the upstream Lua MOVE or CHANGE (OLD=<rev>)
  citation-audit [gofile]
                        FILE=<go file> or SWEEP=1: first and last Lua line of every //pob: range
  extract-output-oracle <dumpsDir> <out.jsonl>
                        the committed accuracy oracle, built from PoB's per-build dumps
  oracle-bin [--check] <oracle.jsonl.gz>
                        the .orb side-car beside a committed oracle blob: the same records in a
                        columnar binary form that needs no JSON decode. GENERATED FROM the
                        .jsonl.gz and keyed by its SHA-256, so a stale one costs time, never
                        accuracy. --check reports staleness without writing; exit 1 = stale
  pob-dump-diff <dirA> <dirB> [--verbose]
                        would the ORACLE BLOB change between two dump sets? exit 1 = yes
  modal-blob <out.jsonl> <report.txt> <in.jsonl>... [--min-fraction=F]
                        vote N single-process extractions into a MODAL oracle blob
  pob-citations [--show <cite> | --selftest]
                        verify the //pob: markers and the port manifest; exit 1 = the gate FAILS
  omission-reasons [--selftest]
                        gate the provenance markers on generated tables' omission reasons
  port-diff             OUR Go beside the PINNED Lua; one of NAME=/SYMBOL=/FUNC=/LUA=/GREP=
  citation-drift-diff [FILTER] [--selftest]
                        for every CHANGED citation, the upstream delta (OLD=<ref>, FILE=<path>)
  config-tools <extract|inventory|compare|splice> [--pob D] [--repo D] [--corpus D] [--var N]...
                        the configs.go porting slices; --dry-run for splice
  pob-drift --candidate DIR [--pin D] [--top N] [--details] [--fix [--dry-run]] [--selftest]
                        every //pob: citation against a CANDIDATE PoB: the upgrade go/no-go
  dataraw-mirror [--rows N] [--selftest]
                        every hand-transcribed data/raw table against the PoB Lua it came from
  mod-scalability-emit <dump.json> <out.go>
  item-mod-stat-order-emit <dump.json> <out.go>
  skill-minion-uses-emit <dump.json> <out.go>
  skill-add-skill-types-emit <dump.json> <out.go>
  skill-has-global-effect-emit <dump.json> <out.go>
  skill-minion-lists-emit <dump.json> <out.go>
  skill-hidden-levels-emit <dump.json> <out.go>
  tattoo-passives-emit <dump.json> <out.go>
  flavour-text-emit <dump.json> <out.go>
  custom-mod-browser-emit <dump.json> <out.go>
  stat-descriptions-emit <dump.json> <out.go>
  skill-text-emit <dump.json> <out.go>
  item-craft-data-emit <dump.json> <out.go>
  item-db-emit <dump.json> <out.go>
  pantheons-emit <dump.json> <out.go>
  cluster-jewels-emit <dump.json> <out.go>
  legion-passives-emit <dump.json> <out.go>
  boss-skills-emit <dump.json> <out.go>
                        oracle/*_emit.py: one generated data/raw table each, from a PoB dump.
                        ⚠️ LISTED SEPARATELY ON PURPOSE — joined by slashes on one line, only the
                        FIRST is in the left column, and the other three read as undocumented to
                        anyone scanning for a name
  skill-flags-emit <dump.json> <out.go>
                        the three skill tables at once: baseFlags, addFlags and the identity row
  gems-emit <dump.json> <out.go> <builds/parser.go>
                        data/raw/gems.go; the id-rewrite table is PARSED from builds/parser.go
  tree-tables-emit <ours.json> <pob.json> <missing.go> <fixups.go>
                        calculator's two tree-diff tables: missing nodes and stat-text fixups
  skill-base-mods-emit <dump.json> <out.go> [calculator/types.go]
                        data/raw/skill_base_mods.go; reads SkillData's FIELD NAMES from types.go
  skill-stat-maps-emit <dump.json> <out.go> [data/raw/skill_stat_map.go]
                        the PER-SKILL stat maps; reads the GLOBAL map to count shadowing entries
  skill-stat-map-global-emit <dump.json> <out.go>
                        the GLOBAL fallback map (SINGULAR); shares every mod expression above
  skill-base-mods-verify <lua.json> <go.json>
                        round-trip: the COMPILED table vs an independent re-parse. 1 = a finding
  skill-stat-map-verify <lua.json> <go.json>
                        the same round-trip for the PER-SKILL stat maps. 1 = a finding
  pob-pin [--record]    assert the PoB checkout IS the tree the oracle was built against.
                        CONTENT is the assertion; the git SHA is only reported
  fma-census            every multiply-add the compiler fuses in the calculation packages on
                        arm64 and amd64 v3; fails on any site not on the allow-list
  regen-fixtures [name ...]
                        regenerate the eight testdata/oracle fixtures from the PINNED PoB.
                        Needs the DETERMINISTIC luajit and setarch -R; neither is a default
  modal-oracle <dumps-base> <runs> <out.jsonl.gz> [workdir]
                        extract N dump dirs and vote them into ONE modal blob; the report beside
                        it carries the disagreement rate and is as much the point as the blob
  affected-tests        [--dry] [go test args...] BASE=<ref>: go test only the packages the change
                        since BASE (default merge-base dev) and the working tree can reach
  test-tiers fast|slow [go test args] | record <log>...
                        the operator suite split at 10s per test (scripts/gopob/operator_slow_tests.txt);
                        fast names any test that crossed the line, record rewrites the list
  nice <command> [args...]
                        run a command at low CPU and IO priority; idempotent across the five
                        nesting layers via GOPOB_NICED. NICE=0 disables, NICE=<n> overrides
  probe-determinism [probe] [runs]
                        is a fixture probe BYTE-reproducible across processes? Starts with a
                        CANARY, because "1 distinct md5" is also what a broken harness prints
  build-luajit-det [--verify]
  require-det-luajit [<task name>]
                        build a LuaJIT whose pairs() order is identical in EVERY process, and
                        prove it. setarch -R is the other half and this cannot enforce it
  repin                 re-measure every assertDivergesFromOracle pin on the CURRENT tree and
                        rewrite it. CHECK=1 reports without writing. Never promotes a CLOSED pin
  test-cache-key        POE_VERSION must be part of go test's result-cache key. Asserted END TO
                        END, by observing go test's own cached/uncached verdict over three runs
  instruments           EVERY instrument, in order, one consolidated report. NOT-RUN is its own
                        class; INSTRUMENT_TIMINGS=0 keeps the clock out of stdout
  regen-data [--check] [--pob DIR] [--only NAME]
                        regenerate every generated data table from the PINNED PoB. --check
                        regenerates to a temp tree and FAILS if a committed artifact differs
  corpus-dump-ref <ref> <out>
                        dump an arbitrary commit's corpus output WITHOUT touching the working
                        tree: it checks the ref out into a throwaway worktree instead
  dump-shard <pob-dir> <repo-root> <builds-dir> <out-dir> <shards>
                        ONE dump, N-way sharded, into one directory — a fast first accuracy
                        number. SOUND only with the DETERMINISTIC luajit; otherwise a MOSAIC
  dump-alone <pob-dir> <repo-root> <builds-dir> <committed-dir> <out-dir> <list-file>
                        dump each listed build ALONE, one process each, and diff its output
                        against a committed dump — names the oracle's order-dependent floor
  dump-parallel <pob-src-parent> <repo-root> <builds-dir> <out-base> <runs>
                        N independent full-corpus PoB dumps at once, each in its own netns.
                        REFUSES before creating anything if a target directory is non-empty
  source-census --our-dump F --stat S [--stat S]... [--dumps D] [--blob B]
                        which SOURCE CLASSES are enriched where one stat is wrong. The only
                        instrument here that can see the OVER-APPLICATION class. Read NORM, not raw
  pob-data-tables-stamp <reference.json|reference-dir>
                        stamp a DATA-table reference with the pin it was dumped from, so the
                        drift gate finds it by CONTENT rather than by a hardcoded filename
  core-constants-emit <dump.json> <out.go>
                        data/constants_generated.go — the VALUES PoB owns; constants.go keeps the
                        TYPES and the Go-native remainder
  minion-data-emit <dump.json> <out.go> <minions|spectres>
                        data/raw/minion_data.go and spectre_data.go — ONE emitter, because they
                        are ONE table upstream
  skill-weapon-types-emit   (dump on STDIN, Go on STDOUT)
  skill-parts <out.go>  PoB's hand-written per-skill parts tables, scanned from POB_DIR's Lua
  extract-calcs-damage <dumpsDir> <out.jsonl>
                        the CALCS-mode damage side-car — extract-output-oracle's sibling
  tree-divergence <ours.json> <pob.json>
                        go-pob-data's passive tree against the pin's: what is UNCOVERED
  task-test-refs [--update-names]
                        every "go test -run <Name>" in the tooling must name a test that EXISTS
                        (SELFTEST=1 plants a dead one; GOPOB_BIN names the gate-versions emitter)
  residual-map          WL=<worklist dir>: where the residual IS -- by family, actor and build.
                        Family totals are UPPER BOUNDS; one root cause counts once per CONSUMER
  comment-strip         [--apply] [--max-prose=N] [--samples=N] [--show=CLASS] [--drop-lua-code] [--keep-go-code] <dir>...:
                        remove go-pob's own prose; keeps markers, PoB's comments, commented Lua
  residual-clusters     MATRIX=<file>: cluster the residual by which builds fail together
  stat-overlap          STATS="A B C" [OUT=<dir>]: do these stats fail on the SAME BUILDS -- the
                        only instrument that separates a downstream aggregate from an independent
                        defect. Containment is a BOUND, not a cause
  corpus-grep           PATTERN=<mod line> [LUA=] [NAMES=]: size a population from the INPUT XML.
                        Handles the newline-wrapped and &apos;-escaped forms a hand grep misses
  coverage-census --profile F --funcs F --root D [--overlay F] [--list] [--json F]
                        which blocks never executed. Track the RATIO: the denominator grows
  coverage-features --census J [--pool DIR] [--shared DIR] [--uniques DIR] [--tsv F] [--grep RE|--grep-file F]
                        the census's dead blocks grouped into PoB features (parser row, config,
                        trigger case, marker), sized on the corpus and on a character-JSON pool
  classify-tree-sources classify a build's tree-sourced mods -- TATTOO, LEGION, jewel conqueror
  tag-census-check [dumpsDir]
                        per-actor mod/tag totals over a dump set. ⚠️ A dump set from another
                        league reports every mod as diverging, which looks like the defect
  mk-drop-cases <drops.tsv> <out.json> [n]
                        parser cases from dropped mod lines. SYMMETRIC: the same wrapper text
                        goes to PoB and to our parser, so a difference is a difference in GRAMMAR
  chunk-init <generated.go> <Name>
                        split a huge literal into buildX() + xPartN() so data/raw.init clears Go
                        1.26's 65536-block wasm cap. WIRED as regen-data's post= step
  zero-value-guard-sweep
                        Go zero-value guards standing in for a PRESENCE test, where Lua
                        distinguishes nil from 0. A SYNTAX question: runtime cannot help
`

func main() {
	if len(os.Args) > 1 && os.Args[1] == "nice" {
		os.Exit(niceRun(os.Args[2:]))
	}
	toolchain.Nice()
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(64)
	}
	switch os.Args[1] {
	case "kw-bits-parity":
		os.Exit(kwBitsParity())
	case "triage-drops":
		os.Exit(triageDrops(os.Args[2:]))
	case "table-row-census":
		os.Exit(tableRowCensus(os.Args[2:]))
	case "optional-token-census":
		os.Exit(optionalTokenCensus(os.Args[2:]))
	case "fold-progress":
		os.Exit(foldProgress(os.Args[2:]))
	case "citation-triage":
		os.Exit(citationTriage(os.Args[2:]))
	case "citation-drift":
		os.Exit(citationDrift(os.Args[2:]))
	case "citation-audit":
		os.Exit(citationAudit(os.Args[2:]))
	case "stat-overlap":
		os.Exit(statOverlap(os.Args[2:]))
	case "classify-tree-sources":
		os.Exit(classifyTreeSources(os.Args[2:]))
	case "affected-tests":
		os.Exit(affectedTests(os.Args[2:]))
	case "test-tiers":
		os.Exit(testTiers(os.Args[2:]))
	case "corpus-grep":
		os.Exit(corpusGrep(os.Args[2:]))
	case "coverage-census":
		os.Exit(coverageCensus(os.Args[2:]))
	case "coverage-features":
		os.Exit(coverageFeatures(os.Args[2:]))
	case "residual-map":
		os.Exit(residualMap(os.Args[2:]))
	case "comment-strip":
		os.Exit(commentStrip(os.Args[2:]))
	case "chunk-init":
		os.Exit(chunkInit(os.Args[2:]))
	case "mk-drop-cases":
		os.Exit(mkDropCases(os.Args[2:]))
	case "residual-clusters":
		os.Exit(residualClusters(os.Args[2:]))
	case "zero-value-guard-sweep":
		os.Exit(zeroValueGuardSweep(os.Args[2:]))
	case "oracle-bin":
		os.Exit(oracleBinEmit(os.Args[2:]))
	case "extract-output-oracle":
		os.Exit(extractOutputOracle(os.Args[2:]))
	case "config-tools":
		os.Exit(configTools(os.Args[2:]))
	case "citation-drift-diff":
		os.Exit(citationDriftDiff(os.Args[2:]))
	case "port-diff":
		os.Exit(portDiff(os.Args[2:]))
	case "omission-reasons":
		os.Exit(omissionReasons(os.Args[2:]))
	case "pob-citations":
		os.Exit(pobCitations(os.Args[2:]))
	case "pob-drift":
		os.Exit(pobDrift(os.Args[2:]))
	case "dataraw-mirror":
		os.Exit(dataRawMirror(os.Args[2:]))
	case "task-test-refs":
		os.Exit(taskTestRefs(os.Args[2:]))
	case "skill-flags-emit":
		os.Exit(skillFlagsEmit(os.Args[2:]))
	case "gems-emit":
		os.Exit(gemsEmit(os.Args[2:]))
	case "tree-tables-emit":
		os.Exit(treeTablesEmit(os.Args[2:]))
	case "skill-base-mods-emit":
		os.Exit(skillBaseModsEmit(os.Args[2:]))
	case "skill-stat-maps-emit":
		os.Exit(skillStatMapEmit(os.Args[2:]))
	case "skill-stat-map-global-emit":
		os.Exit(skillStatMapGlobalEmit(os.Args[2:]))
	case "skill-base-mods-verify":
		os.Exit(skillBaseModsVerify(os.Args[2:]))
	case "skill-stat-map-verify":
		os.Exit(skillStatMapVerify(os.Args[2:]))
	case "pob-pin":
		os.Exit(pobPin(os.Args[2:]))
	case "fma-census":
		os.Exit(fmaCensus(os.Args[2:]))
	case "pob-data-tables-stamp":
		os.Exit(pobDataTablesStamp(os.Args[2:]))
	case "source-census":
		os.Exit(sourceCensus(os.Args[2:]))
	case "dump-parallel":
		os.Exit(dumpParallel(os.Args[2:]))
	case "dump-alone":
		os.Exit(dumpAlone(os.Args[2:]))
	case "dump-shard":
		os.Exit(dumpShard(os.Args[2:]))
	case "corpus-dump-ref":
		os.Exit(corpusDumpRef(os.Args[2:]))
	case "test-cache-key":
		os.Exit(testCacheKey(os.Args[2:]))
	case "instruments":
		os.Exit(instrumentsMain(os.Args[2:]))
	case "regen-data":
		os.Exit(regenData(os.Args[2:]))
	case "repin":
		os.Exit(repin(os.Args[2:]))
	case "build-luajit-det":
		os.Exit(buildLuajitDet(os.Args[2:]))
	case "require-det-luajit":
		os.Exit(requireDetLuajit(os.Args[2:]))
	case "probe-determinism":
		os.Exit(probeDeterminism(os.Args[2:]))
	case "modal-oracle":
		os.Exit(modalOracle(os.Args[2:]))
	case "regen-fixtures":
		os.Exit(regenFixtures(os.Args[2:]))
	case "core-constants-emit":
		os.Exit(coreConstantsEmit(os.Args[2:]))
	case "minion-data-emit":
		os.Exit(minionDataEmit(os.Args[2:]))
	case "skill-parts":
		os.Exit(skillParts(os.Args[2:]))
	case "extract-calcs-damage":
		os.Exit(extractCalcsDamage(os.Args[2:]))
	case "tree-divergence":
		os.Exit(treeDivergence(os.Args[2:]))
	case "mod-scalability-emit":
		os.Exit(modScalabilityEmit(os.Args[2:]))
	case "item-mod-stat-order-emit":
		os.Exit(itemModStatOrderEmit(os.Args[2:]))
	case "skill-minion-uses-emit":
		os.Exit(oracleSkillMinionUsesEmit(os.Args[2:]))
	case "skill-add-skill-types-emit":
		os.Exit(oracleSkillAddSkillTypesEmit(os.Args[2:]))
	case "skill-has-global-effect-emit":
		os.Exit(oracleSkillHasGlobalEffectEmit(os.Args[2:]))
	case "skill-minion-lists-emit":
		os.Exit(oracleSkillMinionListsEmit(os.Args[2:]))
	case "skill-hidden-levels-emit":
		os.Exit(oracleSkillHiddenLevelsEmit(os.Args[2:]))
	case "tattoo-passives-emit":
		os.Exit(oracleTattooPassivesEmit(os.Args[2:]))
	case "flavour-text-emit":
		os.Exit(oracleFlavourTextEmit(os.Args[2:]))
	case "custom-mod-browser-emit":
		os.Exit(oracleCustomModBrowserEmit(os.Args[2:]))
	case "stat-descriptions-emit":
		os.Exit(oracleStatDescriptionsEmit(os.Args[2:]))
	case "skill-text-emit":
		os.Exit(oracleSkillTextEmit(os.Args[2:]))
	case "item-craft-data-emit":
		os.Exit(oracleItemCraftDataEmit(os.Args[2:]))
	case "item-db-emit":
		os.Exit(oracleItemDBEmit(os.Args[2:]))
	case "pantheons-emit":
		os.Exit(oraclePantheonsEmit(os.Args[2:]))
	case "cluster-jewels-emit":
		os.Exit(oracleClusterJewelsEmit(os.Args[2:]))
	case "legion-passives-emit":
		os.Exit(oracleLegionPassivesEmit(os.Args[2:]))
	case "boss-skills-emit":
		os.Exit(oracleBossSkillsEmit(os.Args[2:]))
	case "skill-weapon-types-emit":
		os.Exit(oracleSkillWeaponTypesEmit(os.Args[2:]))
	case "modal-blob":
		os.Exit(modalBlob(os.Args[2:]))
	case "pob-dump-diff":
		os.Exit(pobDumpDiff(os.Args[2:]))
	case "tag-census-check":
		os.Exit(tagCensusCheck(os.Args[2:]))
	case "data-fields":
		os.Exit(dataFields(os.Args[2:]))
	case "gate-versions":
		os.Exit(gateVersions(os.Args[2:]))
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "REFUSING: unknown subcommand %q\n\n%s", os.Args[1], usage)
		os.Exit(64)
	}
}

func gateVersions(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "REFUSING: gate-versions needs exactly one of --sh or --tsv; "+
			"a default output format is how a caller ends up parsing the wrong one")
		return 64
	}
	switch args[0] {
	case "--sh":
		fmt.Print(gateversions.Shell())
	case "--tsv":
		fmt.Print(gateversions.TSV())
	default:
		fmt.Fprintf(os.Stderr, "REFUSING: unknown gate-versions format %q\n", args[0])
		return 64
	}
	return 0
}
