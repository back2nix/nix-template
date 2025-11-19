{
  description = "Micro-Frontend: Shell + Greeter Service (gRPC)";

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
        };

        # Импортируем наш билдер
        buildApp = import ./lib/build.nix;
      in
      {
        packages = rec {
          # --- 1. SHELL (HOST) :8080 ---
          shell = buildApp {
            inherit pkgs lib gomod2nix;
            name = "shell";
            srcBackend = ./shell/backend;
            srcFrontend = ./shell/frontend;
            port = "8080";
          };

          # --- 2. GREETER (REMOTE) :8081 ---
          greeter = buildApp {
            inherit pkgs lib gomod2nix;
            name = "greeter";
            srcBackend = ./services/greeter/backend;
            srcFrontend = ./services/greeter/frontend;
            port = "8081";
          };

          default = shell;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go gomod2nix gopls
            nodejs_20 yarn
            protobuf protoc-gen-go protoc-gen-go-grpc grpcurl
            jq
          ];
          shellHook = ''
             echo "🛠  Micro-Frontend Dev Environment"
             echo "Run 'go run .' in separate terminals for shell and greeter"
          '';
        };
      }
    );
}
