{
  description = "Microservices: Gateway (Envoy) + Services (Go/DDD)";

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
          overlays = [ gomod2nix.overlays.default ];
          config.allowUnfree = true;
        };

        buildService = import ./lib/build-service.nix;
        # buildBackendOnly более не нужен для Gateway, но может пригодиться для других
        buildEnvoyGateway = import ./lib/build-envoy-gateway.nix;

        # --- SERVICES ---

        # Gateway теперь собирается через Envoy
        gatewayPkg = buildEnvoyGateway {
          inherit pkgs;
          name = "gateway";
          port = "8080";
        };

        shellPkg = buildService {
          inherit pkgs gomod2nix;
          name = "shell";
          srcBackend = ./shell/backend;
          srcFrontend = ./shell/frontend;
          port = "3000";
          yarnHash = "sha256-1/c8dhDK/63cUSJlB0GAn9aCSeejZrMb/3yq5EZRak0=";
        };

        greeterPkg = buildService {
          inherit pkgs gomod2nix;
          name = "greeter";
          srcBackend = ./services/greeter/backend;
          srcFrontend = ./services/greeter/frontend;
          port = "50051";
          yarnHash = "sha256-1/c8dhDK/63cUSJlB0GAn9aCSeejZrMb/3yq5EZRak0=";
        };

        projectPackages = {
          gateway = gatewayPkg;
          shell = shellPkg;
          greeter = greeterPkg;
        };

      in
      {
        # --- PACKAGES ---
        packages = projectPackages // {
          all = pkgs.symlinkJoin {
            name = "all-services";
            paths = [ gatewayPkg shellPkg greeterPkg ];
          };
          default = gatewayPkg;
        };

        # --- CHECKS ---
        checks = {
          k3s-integration = import ./tests/k3s-test.nix {
            inherit system pkgs;
            packages = projectPackages;
          };
        };

        # --- DEV SHELL ---
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gomod2nix.packages.${system}.default
            gopls
            nodejs_20
            yarn
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            grpcurl
            just
            jq
            kubectl
            k3s
            # Добавляем envoy в dev shell, чтобы можно было запускать локально
            envoy
            gettext # для envsubst
          ];

          shellHook = ''
            echo "🛠  Microservices Dev Environment (Envoy Enabled)"
            echo "Gateway: :8080 (Envoy)"
            echo "Run: just dev-all"
          '';
        };
      }
    );
}
