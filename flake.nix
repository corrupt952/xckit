{
  description = "CLI tool for managing Xcode String Catalogs (.xcstrings)";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }:
    let
      systems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      # Prefer the commit hash over a hand-maintained version: it can never
      # go stale, and `nix run github:corrupt952/xckit` always builds main.
      version = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "xckit";
            inherit version;
            src = pkgs.lib.cleanSource self;
            vendorHash = "sha256-vU5ZB9c4xqvWEk7NZkchaUlGS0mzDi5YHbe/Apq+FaU=";
            # Keep in sync with .goreleaser.yml / Makefile LDFLAGS.
            ldflags = [ "-s" "-w" "-X" "xckit/command.Version=${version}" ];
            meta.mainProgram = "xckit";
          };
        });

      checks = forAllSystems (system: {
        default = self.packages.${system}.default;
      });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShellNoCC {
            packages = with pkgs; [
              go
              gopls
              gotools
              golangci-lint
              goreleaser
            ];
          };
        });
    };
}
