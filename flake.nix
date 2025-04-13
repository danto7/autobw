{
  description = "development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    unstable.url = "github:NixOS/nixpkgs";
    utils.url = "github:numtide/flake-utils";
    nix-pre-commit.url = "github:jmgilman/nix-pre-commit";
  };

  outputs =
    {
      nixpkgs,
      unstable,
      utils,
      self,
      nix-pre-commit,
      ...
    }:
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        unstablePkgs = import unstable {
          inherit system;
          # bitwarden-cli is broken. Fix: https://github.com/NixOS/nixpkgs/issues/339576#issuecomment-2574076670
          overlays = [
            (final: prev: {
              bitwarden-cli = prev.bitwarden-cli.overrideAttrs (oldAttrs: {
                nativeBuildInputs = (oldAttrs.nativeBuildInputs or [ ]) ++ [ prev.llvmPackages_18.stdenv.cc ];
                stdenv = prev.llvmPackages_18.stdenv;
              });
            })
          ];
        };
        config = {
          repos = [
            {
              repo = "local";
              hooks = [
                # {
                #   id = "nixpkgs-fmt";
                #   entry = "${pkgs.nixpkgs-fmt}/bin/nixpkgs-fmt";
                #   language = "system";
                #   files = "\\.nix";
                # }
                {
                  id = "version-lint";
                  entry = "bash ${builtins.toFile "version-lint.sh" ''
                    set -eo pipefail
                    git_tag="$(git tag --points-at HEAD)"
                    if test -z "$git_tag"; then
                      echo "version-lint: No new version found on this commit." >&2
                      exit 0
                    fi
                    flake_version="$(grep -E "[0-9]+\.[0-9]+\.[0-9]+" flake.nix --only-matching)"
                    if test "$git_tag" != "v$flake_version"; then
                      echo "version-lint: Version in flake.nix ($flake_version) does not match git tag ($git_tag)" >&2
                      exit 1
                    fi
                  ''}";
                  language = "system";
                  stages = [ "pre-push" ];
                }
              ];
            }
          ];
        };
      in
      with nixpkgs.legacyPackages.${system};
      {
        devShells = {
          default = pkgs.mkShell {
            shellHook =
              (nix-pre-commit.lib.${system}.mkConfig {
                inherit pkgs config;
              }).shellHook;
            packages = with pkgs; [
              # kubectl
              # minikube
              # docker-client
              go
              goreleaser
              unstablePkgs.bitwarden-cli
            ];
          };
        };
        defaultPackage = buildGoModule rec {
          pname = "autobw";
          version = "git";

          src = builtins.path {
            path = ./.;
            name = "${pname}-src";
            filter = path: type: path != ".git";
          };
          ldflags = [
            "-X main.bwBinary=${unstablePkgs.bitwarden-cli}/bin/bw"
            "-X main.version=${version} -X main.commit=${self.rev or "dirty"} -X main.date=unknown"
            "-X state.keychainSuffix=''"
          ];

          vendorHash = "sha256-gnbZiWGWoMuZgs4IssDIQdHjzT2biPlyjdhBxz3wN0o=";

          meta = {
            description = "Command line tool to manage Bitwarden cli sessions in the keychain";
            homepage = "https://github.com/danto7/autobw";
            license = lib.licenses.mit;
            maintainers = with lib.maintainers; [ danto7 ];
          };
        };
        packages.bitwarden-cli = unstablePkgs.bitwarden-cli;
      }
    );
}
