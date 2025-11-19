{
  description = "Go (gRPC) + Vue (Yarn)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, flake-utils, gomod2nix }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
          config.allowUnfree = true;
        };
      in
      {
        packages = rec {
          # --- BACKEND ---
          backend = pkgs.callPackage ./backend/default.nix { };

          # --- FRONTEND (Yarn) ---
          # Если фронт не нужен — закомментируй следующую строку и раскомментируй null
          frontend = pkgs.callPackage ./frontend/default.nix { };
          # frontend = null;

          # --- FULL APP ---
          default = pkgs.callPackage ./default.nix {
            inherit backend frontend;
          };

          docker = pkgs.callPackage ./backend/docker.nix { app = default; };
        };

        devShells.default = pkgs.callPackage ./shell.nix { };
      }
    );
}
