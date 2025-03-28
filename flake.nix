{
  description = "development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    unstable.url = "github:NixOS/nixpkgs";
    utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      unstable,
      utils,
      self,
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
      in
      with nixpkgs.legacyPackages.${system};
      {
        devShells = {
          default = pkgs.mkShell {
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
          version = "0.1.1";

          src = builtins.path {
            path = ./.;
            name = "${pname}-src";

          };
          ldflags = [
            "-X main.bwBinary=${unstablePkgs.bitwarden-cli}/bin/bw"
            "-X main.version=${version} -X main.commit=${self.rev or "dirty"} -X main.date=unknown"
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
