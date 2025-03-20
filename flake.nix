{
  description = "development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, utils, ... }:
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
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
            ];
          };
        };
        defaultPackage = buildGoModule rec {
          pname = "autobw";
          version = "0.3.4";

          src = builtins.path {
            path = ./.;
            name = "${pname}-src";
          };

          vendorHash = "sha256-kUZ2CxKfx/QeKXxix24Ld9lK50MEENuH5HS9gWY8zZo=";

          meta = {
            description = "Command line tool to manage Bitwarden cli sessions in the keychain";
            homepage = "https://github.com/danto7/autobw";
            license = lib.licenses.mit;
            maintainers = with lib.maintainers; [ danto7 ];
          };
        };
      }
    );
}
