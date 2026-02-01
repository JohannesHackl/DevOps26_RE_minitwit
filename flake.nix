{
  description = "MiniTwit FHS environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};

      fhs = pkgs.buildFHSUserEnv {
        name = "minitwit-env";
        targetPkgs = pkgs: with pkgs; [
          sqlite
          ##Python
          (python312.withPackages (ps: with ps; [
            numpy
            requests
            pandas
            flask
            pip
            virtualenv
            werkzeug
          ]))
          gcc
          gnumake

        ];
        runScript = "bash";
      };
    in
    {
      devShells.${system}.default = fhs.env;

      # Alternative: direct app to run the binary
      apps.${system}.flag_tool = {
        type = "app";
        program = "${fhs}/bin/minitwit-env -c ./flag_tool";
      };
    };
}
