# Contributing to Path of Building

## Table of contents
1. [Reporting bugs](#reporting-bugs)
2. [Requesting features](#requesting-features)
3. [Contributing code](#contributing-code)
4. [Setting up a development installation](#setting-up-a-development-installation)
5. [Setting up a development environment](#setting-up-a-development-environment)
6. [Testing](#testing)
7. [Linting](#linting)

## Reporting bugs

### Before creating an issue:
* Check that the bug hasn't been reported in an existing issue. View similar issues to the left of the submit button.
* Make sure you are running the latest version of the program. Click "Check for Update" in the bottom left corner.
* If you've found an issue with offence or defence calculations, make sure you check the breakdown for that calculation in the Calcs tab to see how it is being performed, as this may help you find the cause.

### When creating an issue:
* Select the "Bug Report" issue template and fill out all fields.
* Please provide detailed instructions on how to reproduce the bug, if possible.
* Provide a build share code for a build that is affected by the bug, if possible.
  In the "Import/Export Build" tab, click "Generate", then "Share" and add the link to your post.

Build share codes allow us to reproduce bugs much more quickly.

## Requesting features
Feature requests are always welcome. Note that not all requests will receive an immediate response.

### Before submitting a feature request:
* Check that the feature hasn't already been requested. Look at all issues with titles that might be related to the feature.
* Make sure you are running the latest version of the program, as the feature may already have been added. Click "Check for Update" in the bottom left corner.

### When submitting a feature request:
* Select the "Feature Request" issue template and fill out all fields.
* Be specific! The more details, the better.
* Small requests are fine, even if it's just adding support for a minor modifier on a rarely-used unique.

## Contributing code

### Before submitting a pull request:
* Familiarise yourself with the code base [here](docs/rundown.md) to get you started.
* There is a [Discord](https://discordapp.com/) server for **active development** on the fork and members are happy to answer your questions there.
  If you are interested in joining, send a private message to any of **Cinnabarit#1341**, **LocalIdentity#9871**, **Yamin#5575** and we'll send you an invitation.

### When submitting a pull request:
* **Pull requests must be created against the `dev` branch**, as all changes to the code are staged there before merging to `main`.
* Make sure that the changes have been thoroughly tested!

## Setting up a development installation
Note: This tutorial assumes that you are already familiar with Git.

Clone the repository using this command:
```shell
git clone -b dev https://github.com/Vilsol/go-pob.git
```

### Backend (Go)

We require you to use Go version 1.19 or above.

To build the WASM binary, you can use the following command:

```shell
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -v -o frontend/static/go-pob.wasm ./wasm
```

To re-generate any new typings that have been exposed from Go, you can use this:
```shell
go generate -tags tools -x ./...
```

### Frontend (Svelte)

We require NodeJS version `18.4.0` which can either be installed via `nvm` or via installer.

We also use the [`pnpm`](https://pnpm.io/) package manager.

You can install all frontend dependencies using:
```shell
pnpm i
```

And then start the dev server via:
```shell
pnpm run dev
```

## Setting up a development environment

Note: This tutorial assumes that you are already familiar with the development tool of your choice.

If you want to use a text editor, [Visual Studio Code](https://code.visualstudio.com/) (proprietary) is recommended.
All you need are the [Go](https://marketplace.visualstudio.com/items?itemName=golang.go) and [Svelte](https://marketplace.visualstudio.com/items?itemName=svelte.svelte-vscode) extensions.

If you want to use an IDE instead, [GoLand](https://www.jetbrains.com/go/) (for backend) and [WebStorm](https://www.jetbrains.com/webstorm/) (for frontend) are recommended.

## Testing

PoB uses the builtin Go testing platform together with [testza](https://github.com/MarvinJWendt/testza) framework.

All tests are run twice, first time in native Go, second time through WASM to ensure that nothing breaks on either deployment.

### Running Native Tests

Simply executing `go test ./...` from the project root directory should run all tests.

### Running WASM Tests

First ensure that you have the appropriate NodeJS version installed. (currently `18.4.0`, easiest managed with `nvm` or equivalent)

Then you should be able to execute all tests using `./.github/wasm_test.sh` script.

## Linting

If any of these linters fail, the CI build will not pass.

### Backend (Go)

The backend is linted using [`golangci-lint`](https://golangci-lint.run/usage/install/). You can execute it via `golangci-lint run`

### Frontend (Svelte)

The frontend is linted using `prettier`, `eslint` and `svelte-check`. You can execute those by using `pnpm run lint` and `pnpm run check`
