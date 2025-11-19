{
  description = "Microservices: Gateway + Services (DDD + Clean Architecture)";

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

        buildService = import ./lib/build-service.nix;

        # Для gateway - только backend
        buildBackendOnly = import ./lib/build-backend-only.nix;
      in
      {
        packages = rec {
          # API Gateway (только backend)
          gateway = buildBackendOnly {
            inherit pkgs gomod2nix;
            name = "gateway";
            srcBackend = ./services/gateway/backend;
            port = "8080";
          };

          # Shell (Host для micro-frontends)
          shell = buildService {
            inherit pkgs gomod2nix;
            name = "shell";
            srcBackend = ./shell/backend;
            srcFrontend = ./shell/frontend;
            port = "3000";
            yarnHash = "sha256-1/c8dhDK/63cUSJlB0GAn9aCSeejZrMb/3yq5EZRak0="; # hash для shell
          };

          # Greeter Service
          greeter = buildService {
            inherit pkgs gomod2nix;
            name = "greeter";
            srcBackend = ./services/greeter/backend;
            srcFrontend = ./services/greeter/frontend;
            port = "50051";
            yarnHash = "sha256-1/c8dhDK/63cUSJlB0GAn9aCSeejZrMb/3yq5EZRak0="; # hash для greeter
          };

          # Собираем все вместе для docker-compose или kubernetes
          all = pkgs.symlinkJoin {
            name = "all-services";
            paths = [ gateway shell greeter ];
          };

          default = gateway;
        };

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
            # для миграций БД
            migrate
            # для тестирования
            golangci-lint
          ];

          shellHook = ''
            echo "🛠  Microservices Dev Environment"
            echo "Gateway: :8080"
            echo "Shell:   :3000"
            echo "Greeter: :50051 (gRPC), :8081 (HTTP)"
            echo ""
            echo "Run: just dev-all"
          '';
        };
      }
    );
}
