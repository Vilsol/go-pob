{
  description = "go-pob: Path of Building, ported to Go, as a desktop app";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      inherit (nixpkgs) lib;
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = f: lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      version = "0-unstable-${
        lib.concatStringsSep "-" [
          (lib.substring 0 4 self.lastModifiedDate)
          (lib.substring 4 2 self.lastModifiedDate)
          (lib.substring 6 2 self.lastModifiedDate)
        ]
      }";

      # what the package reads: the Go modules, the frontend and the desktop host
      src = lib.fileset.toSource {
        root = ./.;
        fileset = lib.fileset.difference ./. (
          lib.fileset.unions (
            map lib.fileset.maybeMissing [
              ./.github
              ./docs
              ./nix
              ./oracle
              ./scripts
              ./testdata
              ./flake.nix
              ./flake.lock
              ./desktop/bin
              ./frontend/build
              ./frontend/node_modules
              ./frontend/.svelte-kit
            ]
          )
        );
      };

      overlay = final: _: {
        go-pob = final.callPackage ./nix/package.nix { inherit src version; };
      };
    in
    {
      overlays.default = overlay;

      packages = forAllSystems (pkgs: rec {
        go-pob = (overlay pkgs pkgs).go-pob;
        default = go-pob;
      });

      apps = forAllSystems (pkgs: {
        default = {
          type = "app";
          program = lib.getExe self.packages.${pkgs.stdenv.hostPlatform.system}.go-pob;
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          inputsFrom = [ self.packages.${pkgs.stdenv.hostPlatform.system}.go-pob ];
          # mise.toml pins exact versions of the same tools (nixpkgs carries the nearest); run-headless.sh needs Xvfb
          # and dbus-run-session
          packages = with pkgs; [
            nodejs_26
            go-task
            golangci-lint
            luajit
            zig_0_13
            nixfmt
            xorg-server
            dbus
          ];
        };
      });

      checks = forAllSystems (pkgs: {
        go-pob = self.packages.${pkgs.stdenv.hostPlatform.system}.go-pob;

        # the nixpkgs template is nix/package.nix as render.sh prints it, and evaluates with this tree as its source
        nixpkgs-template =
          let
            template = ./nix/nixpkgs/pkgs/by-name/go/go-pob/package.nix;
            drv = pkgs.callPackage template { fetchFromGitHub = _: src; };
          in
          pkgs.runCommand "go-pob-nixpkgs-template"
            { drvPath = builtins.unsafeDiscardStringContext drv.drvPath; }
            ''
              version=$(sed -n 's/^  version = "\(.*\)";$/\1/p' ${template})
              hash=$(sed -n '/^    tag = /{n;s/^    hash = "\(.*\)";$/\1/p}' ${template})
              bash ${./nix}/nixpkgs/render.sh "$version" "''${hash:-fake}" | diff ${template} -
              touch $out
            '';
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
