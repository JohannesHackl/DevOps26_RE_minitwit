{
  description = "Minitweet NixOS deployment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    {
      nixosConfigurations = {
        minitweet-db = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            ./nix-hosts/db/configuration.nix
            ./modules/postgres.nix
          ];
        };

        minitweet-api = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            ./nix-hosts/app/configuration.nix
            ./modules/minitwit-app.nix
          ];
        };
      };
    };
}
