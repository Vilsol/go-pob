# Contributing to go-pob

## Table of contents
1. [Reporting bugs](#reporting-bugs)
2. [Requesting features](#requesting-features)
3. [Contributing code](#contributing-code)
4. [Setting up a development installation](#setting-up-a-development-installation)
5. [Setting up a development environment](#setting-up-a-development-environment)
6. [Testing](#testing)
7. [Linting](#linting)
8. [Releasing](#releasing)

## Reporting bugs

### Before creating an issue:
* Check that the bug hasn't been reported in an existing issue. View similar issues to the left of the submit button.
* Make sure you are on the latest version: reload [pob.gg](https://pob.gg) (or [dev.pob.gg](https://dev.pob.gg) for the rolling build).
* If you've found an issue with offence or defence calculations, make sure you check the breakdown for that calculation in the Calcs tab to see how it is being performed, as this may help you find the cause.

### When creating an issue:
* Select the "Bug Report" issue template and fill out all fields.
* Please provide detailed instructions on how to reproduce the bug, if possible.
* Provide a build code for a build that is affected by the bug, if possible.
  On the Import / Export page, generate the build code and paste it (or a pobb.in link) into your post.

Build codes allow us to reproduce bugs much more quickly.

## Requesting features
Feature requests are always welcome. Note that not all requests will receive an immediate response.

### Before submitting a feature request:
* Check that the feature hasn't already been requested. Look at all issues with titles that might be related to the feature.
* Make sure you are on the latest version, as the feature may already have been added.

### When submitting a feature request:
* Select the "Feature Request" issue template and fill out all fields.
* Be specific! The more details, the better.
* Small requests are fine, even if it's just adding support for a minor modifier on a rarely-used unique.

## Contributing code

### Before submitting a pull request:
* Familiarise yourself with the layout in the [README](README.md#layout).
* There is a [Discord](https://discordapp.com/) server for **active development** on the fork and members are happy to answer your questions there.
  If you are interested in joining, send a private message to any of **Cinnabarit#1341**, **LocalIdentity#9871**, **Yamin#5575** and we'll send you an invitation.

### When submitting a pull request:
* **Pull requests must be created against the `dev` branch.** `dev` deploys continuously to [dev.pob.gg](https://dev.pob.gg);
  `main` holds releases, which deploy to [pob.gg](https://pob.gg).
* Make sure that the changes have been thoroughly tested!

## Setting up a development installation
Note: This tutorial assumes that you are already familiar with Git.

Clone the repository using this command:
```shell
git clone -b dev https://github.com/Vilsol/go-pob.git
```

### Windows / Linux / macOS

You only need to install `mise` which is a package manager: https://mise.jdx.dev/getting-started.html

Once you install `mise` you can run `mise install` to install all packages.

Every dev command is a mise task; `mise tasks` lists them. Running them through mise gives everyone,
CI included, the same pinned toolchain.

After you have downloaded everything, you might need to run `mise x -- corepack enable` to be able to use `pnpm`

## Setting up a development environment

Note: This tutorial assumes that you are already familiar with the development tool of your choice.

If you want to use a text editor, [Visual Studio Code](https://code.visualstudio.com/) (proprietary) is recommended.
All you need are the [Go](https://marketplace.visualstudio.com/items?itemName=golang.go) and [Svelte](https://marketplace.visualstudio.com/items?itemName=svelte.svelte-vscode) extensions.

If you want to use an IDE instead, [GoLand](https://www.jetbrains.com/go/) (for backend) and [WebStorm](https://www.jetbrains.com/webstorm/) (for frontend) are recommended.

### Backend (Go)

To build the WASM binary, you can use the following command:

```shell
mise run build-go
```

To re-generate any new typings that have been exposed from Go, you can use this:
```shell
mise run generate
```

If you want those to run continuously while developing, you can use:
```shell
mise run dev-go
```

### Frontend (Svelte)

The frontend resides in the `./frontend`

You can start the dev server via:
```shell
mise run dev-frontend
```

You can build it via:
```shell
mise run build-frontend
```

## Testing

go-pob uses Go's built-in testing package together with the [testza](https://github.com/MarvinJWendt/testza) framework.

All tests are run twice, first time in native Go, second time through WASM to ensure that nothing breaks on either deployment.

### Running Native Tests

Simply executing `mise run test` should run all tests.

### Running the corpus gate

The accuracy gate grades every build in `testdata/corpus-<version>/` against what Path of Building computes for it
(`testdata/oracle-<version>/`). Both are release archives in
[go-pob-testdata](https://github.com/Vilsol/go-pob-testdata), pinned by `scripts/testdata.json`; many unit tests read
corpus builds too, so fetch them once after cloning (and again whenever the pin changes):

```shell
mise run testdata
mise run task -- gate
```

### Running WASM Tests (you probably don't need this)

You should be able to execute all tests using `./.github/wasm_test.sh` script.

## Linting

If any of these linters fail, the CI build will not pass.

### Backend (Go)

The backend is linted using [`golangci-lint`](https://golangci-lint.run/usage/install/).

You can execute it via `mise run lint-go` and format with `mise run format-go`.

### Frontend (Svelte)

The frontend is linted using `prettier` and `eslint`.

You can execute those by using `mise run lint-frontend` and format with `mise run format-frontend`.

## Setting up a PoB reference environment

Regenerating the data tables or the oracle needs a Path of Building checkout of the pinned version (upstream
v2.67.2, recorded in `scripts/pob_pin.json`), at `~/PathOfBuilding` or wherever `POB_DIR` points.
`mise run task -- pob-pin` verifies that the checkout matches the pin.

To debug PoB itself, run it under [EmmyLua](https://github.com/EmmyLua) and add the following to
`.vscode/launch.json` in your PoB checkout, then start the debugger from the VS Code UI:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "type": "emmylua_new",
            "request": "launch",
            "name": "Debug",
            "host": "localhost",
            "port": 9966,
            "ext": [
                ".lua",
                ".lua.txt",
                ".lua.bytes"
            ],
            "ideConnectDebugger": true
        }
    ]
}
```

## Writing code

In general, follow Go's best practices.

If you are using regex, make sure that you precompile it instead of creating a new one every time. (as Go's regex implementation runs in [linear](https://github.com/golang/go/blob/1176052bb40378272cfbe83d873b65fcc2ed8502/src/regexp/regexp.go#L15-L19) time)

### Converting Code from PoB

[CONVERSION_EXAMPLES.md](CONVERSION_EXAMPLES.md)
## Releasing

Releases are made by `.github/workflows/release.yml` on every push to `main`. [semantic-release](https://semantic-release.gitbook.io)
(`.releaserc.json`, pinned in `.github/release/`) picks the version from the conventional commit subjects since the last
`v*` tag. In order:

1. **Version**: `semantic-release --dry-run` works out the next version. Nothing is tagged yet.
2. **Web**: [pob.gg](https://pob.gg) is built and deployed with that version (shown in About go-pob). A push with
   nothing to release deploys `<last release>+<short sha>`.
3. **Desktop** and **Flatpak**: `desktop.yml` and `flatpak.yml`, called with the version, build every package as that
   version, and `update.json` is signed for it.
4. **Publish**: semantic-release runs again, tags `vX.Y.Z`, creates the GitHub release with its notes and attaches
   every package, the `.flatpak` and `update.json`. It refuses any version but the one the packages were built as.

A failed build or deploy stops the run before **Publish**, so nothing is tagged. A push to `dev` deploys
[dev.pob.gg](https://dev.pob.gg) showing the short commit sha, and never tags.

### Version rules

| commit | release |
|---|---|
| `feat:` | minor (0.1.0 → 0.2.0) |
| `fix:`, `perf:` | patch (0.1.0 → 0.1.1) |
| breaking (`feat!:`, a `BREAKING CHANGE:` footer) | minor while the version is 0.x |
| anything else (`chore:`, `docs:`, `ci:`, …) | none |

semantic-release has no 0.x mode, so `.releaserc.json` maps a breaking change to a minor release. To release 1.0.0,
delete that `releaseRules` entry in the same push as a breaking commit.

### The first release (once)

semantic-release starts at 1.0.0 when it finds no tag; go-pob starts at 0.1.0. Before the first push to `main` that
should release, tag the repository's first commit `v0.0.0`:

```shell
git fetch origin main
git tag v0.0.0 "$(git rev-list --max-parents=0 origin/main)"
git push origin v0.0.0
```

The next push to `main` then releases 0.1.0 (the history holds `feat:` commits). Until a `v*` tag exists the release
workflow fails at **Version** rather than release 1.0.0.

### Secrets and variables

| name | kind | for |
|---|---|---|
| `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID` | secrets | deploying pob.gg and dev.pob.gg |
| `UPDATE_PRIVATE_KEY` | secret | signing `update.json` (the PEM `wails3 updater genkey` writes). Losing it strands every installed desktop build |
| `UPDATE_PUBLIC_KEY` | variable | the base64 public key built into release packages. Without both keys, releases ship no `update.json` and no updater |
| `DESKTOP_WINDOWS`, `DESKTOP_MACOS` | variables | `on` builds the Windows and macOS packages |
| `PACKAGE_MAINTAINER` | variable | the deb / rpm maintainer (`Name <email>`; default: the repository owner) |
| `WINDOWS_CERT_PFX`, `WINDOWS_CERT_PASSWORD` | secrets | optional Windows code signing |
| `MACOS_CERT_P12`, `MACOS_CERT_PASSWORD`, `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_PASSWORD` | secrets | optional macOS Developer ID signing and notarisation |

The tag and the release are made with the workflow's own `GITHUB_TOKEN`; no personal access token is needed.

### After a release, by hand

- **Flathub**: the first submission is a pull request to [flathub/flathub](https://github.com/flathub/flathub) (branch
  `new-pr`) with `desktop/build/flatpak/io.github.vilsol.go-pob.yml`, its source switched to `type: git` at the tag and
  commit, and `go-sources.json` / `node-sources.json` regenerated there (`mise run task -- flatpak-sources`). Later
  releases are pull requests to `flathub/io.github.vilsol.go-pob` with the new tag, commit and sources. Add the
  release to the metainfo's `<releases>` in the repository too; CI only adds it to the bundle it builds.
- **nixpkgs**: `mise run task -- nixpkgs-template VERSION=<x.y.z> HASH=<hash>` renders the package for a pull request to
  nixpkgs; [`nix/nixpkgs/README.md`](nix/nixpkgs/README.md) has the steps.
