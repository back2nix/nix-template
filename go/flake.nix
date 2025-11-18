{
  description = "Standard Go Project Template";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, flake-utils, gomod2nix }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            gomod2nix.overlays.default
            (final: prev: {
              # Здесь можно жестко задать версию Go для всего проекта
              # go = prev.go;
              # Используем buildGoApplication из gomod2nix вместо стандартного buildGoModule
              buildGoApplication = prev.buildGoApplication;
            })
          ];
        };
      in
      {
        packages = rec {
          # Основное приложение
          app = pkgs.callPackage ./default.nix { };

          # Docker образ
          docker = pkgs.callPackage ./docker.nix { inherit app; };

          default = app;
        };

        devShells.default = pkgs.callPackage ./shell.nix { };
      }
    );
}
