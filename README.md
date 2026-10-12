# go-pob [![push](https://github.com/Vilsol/go-pob/actions/workflows/push.yml/badge.svg)](https://github.com/Vilsol/go-pob/actions/workflows/push.yml) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/vilsol/go-pob)

A rewrite of Path of Building in Go.

Contributions are welcome! [CONTRIBUTING.md](CONTRIBUTING.md)

Hosted version: [https://pob.gg](https://pob.gg)

Uses data extracted with https://github.com/Vilsol/go-pob-data

## Layout

| path | what |
|---|---|
| `calculator/`, `mod/`, `moddb/`, `data/`, `pob/`, `builds/` | the calculation engine, ported from PoB's Lua and kept structurally parallel to it; `//pob:` markers cite the Lua lines each part ports |
| `wasm/` | the WASM entry point and the functions exposed to the frontend |
| `frontend/` | the SvelteKit frontend (`mise run dev-frontend`) |
| `scripts/gopob/` | the development tool behind most `task` targets: data generators, the oracle tools, the PoB pin check |
| `scripts/gamedatadump/`, `scripts/corpus/` | the game-data dumper and the test-corpus harvester |
| `oracle/` | Lua probes run inside a pinned PoB checkout to produce the reference data under `testdata/` |
| `testdata/` | the build corpus and PoB's own output for every build (fetched by `mise run testdata`) |

## Testing against PoB

`testdata/corpus-<version>/` holds real build exports and `testdata/oracle-<version>/` holds what the pinned
PoB computes for each of them. The corpus and the oracle are release archives in
[go-pob-testdata](https://github.com/Vilsol/go-pob-testdata), pinned by `scripts/testdata.json`; run `mise run testdata`
after cloning (and whenever the pin changes) to fetch them. Many unit tests read corpus builds too.

```shell
mise run test             # unit tests, Go and frontend
mise run task -- gate     # every corpus build against the oracle (needs the fetched data)
```

Regenerating the data tables or the oracle needs a Path of Building checkout at the commit recorded in
`scripts/pob_pin.json` (`task pob-pin` verifies it), at `~/PathOfBuilding` or wherever `POB_DIR` points.

## Credits

Massive thank you to all PoB devs and contributors, this project would be impossible without any of you!
