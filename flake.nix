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
            ./hosts/minitweet-db/configuration.nix
            ./modules/postgres.nix
          ];
        };

        minitweet-api = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            ./hosts/minitweet-api/configuration.nix
            ./modules/minitweet-app.nix
            ./modules/nginx.nix
          ];
        };
      };
    };
}
