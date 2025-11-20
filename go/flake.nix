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
        buildBackendOnly = import ./lib/build-backend-only.nix;

        # --- ОПРЕДЕЛЯЕМ ПАКЕТЫ ЗДЕСЬ (в let блоке) ---
        # Это позволяет безопасно передавать их и в outputs.packages, и в outputs.checks

        gatewayPkg = buildBackendOnly {
          inherit pkgs gomod2nix;
          name = "gateway";
          srcBackend = ./services/gateway/backend;
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

        # Группируем их в объект для удобства передачи
        projectPackages = {
          gateway = gatewayPkg;
          shell = shellPkg;
          greeter = greeterPkg;
        };

      in
      {
        # --- PACKAGES ---
        packages = projectPackages // {
          # Собираем все вместе
          all = pkgs.symlinkJoin {
            name = "all-services";
            paths = [ gatewayPkg shellPkg greeterPkg ];
          };

          default = gatewayPkg;
        };

        # --- CHECKS (TESTS) ---
        checks = {
          k3s-integration = import ./tests/k3s-test.nix {
            inherit system pkgs;
            packages = projectPackages; # Теперь переменная доступна
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
            # для миграций БД
            migrate
            # для тестирования
            golangci-lint
            # утилиты для k8s (полезно в dev shell)
            kubectl
            k3s
          ];

          shellHook = ''
            echo "🛠  Microservices Dev Environment"
            echo "Gateway: :8080"
            echo "Shell:   :3000"
            echo "Greeter: :50051 (gRPC), :8081 (HTTP)"
            echo ""
            echo "Run: just dev-all"
            echo "Test: nix flake check -L"
          '';
        };
      }
    );
}
