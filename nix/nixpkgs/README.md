# go-pob in nixpkgs

`pkgs/by-name/go/go-pob/package.nix` is `../package.nix` as `render.sh` prints it: the same derivation, its source
fetched from `github:Vilsol/go-pob` at a tag. The flake does not use it; `nix flake check` fails when it drifts
from `../package.nix` (`task nixpkgs-template` regenerates it).

## Opening the nixpkgs PR

1. Tag the public repository (`v0.1.0`) and push the tag. `nix run github:Vilsol/go-pob/v0.1.0` should start.
2. In a nixpkgs checkout on a fresh branch off `master`:

   ```shell
   mkdir -p pkgs/by-name/go/go-pob
   <go-pob>/nix/nixpkgs/render.sh 0.1.0 > pkgs/by-name/go/go-pob/package.nix   # prefetches the tag's hash
   nix-build -A go-pob
   nixfmt --check pkgs/by-name/go/go-pob/package.nix
   ./result/bin/go-pob
   ```

   `vendorHash` and the pnpm hash carry over while `go.sum`, `desktop/go.sum` and `frontend/pnpm-lock.yaml` match the
   ones this file was built with; otherwise `nix-build` names the new value.
3. Commit as `go-pob: init at 0.1.0` (one commit, the package alone) and open the PR against `master`, filling in the
   template's checklist (built on x86_64-linux, sandboxed, binary run).

## What the reviewers and CI look at

- `pkgs/by-name` layout: the directory is the attribute name, two-letter shard `go`, a `package.nix` called with
  `callPackage` and no extra arguments, no entry in `all-packages.nix`, nothing referenced outside the directory
  (checked by `nixpkgs-vet`).
- Formatting: `nixfmt` (the treefmt check).
- Evaluation on every platform in `meta.platforms` (Linux only here: the Wails macOS path needs the Apple SDK and cgo)
  and a build on x86_64-linux and aarch64-linux.
- `meta`: `vilsol` is already in `maintainers/maintainer-list.nix`; licenses are `gpl3Only` (go-pob) and `mit` (Path of
  Building's code it ports) — change to `gpl3Plus` if the project means "or later".
- Updates afterwards: `passthru.updateScript` is `nix-update-script`, which the update bot runs on new tags; it bumps
  the version, the source hash, `vendorHash` and the pnpm hash.
