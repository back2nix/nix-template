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
        buildEnvoyGateway = import ./lib/build-envoy-gateway.nix;
        buildImage = import ./lib/build-image.nix;

        # Конфигурация для разных окружений
        envConfig = {
          local = {
            otelEndpoint = "http://localhost:4318/v1/traces";
            gatewayUrl = "http://localhost:8080";
          };
          k8s = {
            otelEndpoint = "http://192.168.3.18:8080/v1/traces";
            gatewayUrl = "http://192.168.3.18:8080";
          };
        };

        # Функция для создания пакетов с определенным окружением
        makePackages = { otelEndpoint, gatewayUrl }: {
          gateway = buildEnvoyGateway {
            inherit pkgs;
            name = "gateway";
            port = "8080";
          };

          shell = buildService {
            inherit pkgs gomod2nix otelEndpoint gatewayUrl;
            name = "shell";
            srcBackend = ./services/shell/backend;
            srcFrontend = ./services/shell/frontend;
            port = "9002";
            yarnHash = "sha256-qDBShQ1JgGqu62YgWJIJn6OLIZvS0aWRNA9LRoyvc7s=";
            basePath = "/";
          };

          landing = buildService {
            inherit pkgs gomod2nix otelEndpoint gatewayUrl;
            name = "landing";
            srcBackend = ./services/landing/backend;
            srcFrontend = ./services/landing/frontend;
            port = "8081";
            yarnHash = "sha256-1/c8dhDK/63cUSJlB0GAn9aCSeejZrMb/3yq5EZRak0=";
            basePath = "/api/landing/";
          };

          chat = buildService {
            inherit pkgs gomod2nix otelEndpoint gatewayUrl;
            name = "chat";
            srcBackend = ./services/chat/backend;
            srcFrontend = ./services/chat/frontend;
            port = "8082";
            yarnHash = "sha256-SXkOoIAtM9ZZ/vnSuX6hUJuOeYPQn1y4qgWAz1rEfZc=";
            basePath = "/api/chat/";
          };

          notification = buildService {
            inherit pkgs gomod2nix;
            name = "notification";
            srcBackend = ./services/notification/backend;
            port = "8085";
          };
        };

        # Пакеты для локальной разработки (default)
        localPackages = makePackages envConfig.local;

        # Пакеты для K8s
        k8sPackages = makePackages envConfig.k8s;

        # --- DOCKER IMAGES (используем K8s пакеты) ---
        dockerImages = {
            # Базовый образ для FAST DEPLOY (Nix Store mounting)
            devBase = pkgs.dockerTools.buildImage {
                name = "dev-base";
                tag = "latest";
                created = "now";
                copyToRoot = [
                    pkgs.bashInteractive # Нужен bash для запуска скриптов
                    pkgs.coreutils
                    pkgs.cacert          # SSL сертификаты
                    pkgs.iana-etc        # /etc/protocols и т.д.
                ];
                config = {
                    Cmd = [ "/bin/bash" ];
                    Env = [
                        "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
                    ];
                };
            };

            gateway = buildImage {
                inherit pkgs;
                servicePkg = k8sPackages.gateway;
                name = "gateway";
                configEnv = [ "GATEWAY_HTTP_PORT=8080" ];
            };
            shell = buildImage {
                inherit pkgs;
                servicePkg = k8sPackages.shell;
                name = "shell";
            };
            landing = buildImage {
                inherit pkgs;
                servicePkg = k8sPackages.landing;
                name = "landing";
            };
            chat = buildImage {
                inherit pkgs;
                servicePkg = k8sPackages.chat;
                name = "chat";
            };
            notification = buildImage {
                inherit pkgs;
                servicePkg = k8sPackages.notification;
                name = "notification";
            };
        };

      in
      {
        # --- PACKAGES ---
        packages = localPackages // {
          inherit dockerImages;

          # Экспортируем K8s пакеты отдельно
          k8s = k8sPackages;

          all = pkgs.symlinkJoin {
            name = "all-services";
            paths = [
              localPackages.gateway
              localPackages.shell
              localPackages.landing
              localPackages.chat
              localPackages.notification
            ];
          };

          default = localPackages.gateway;
        };

        # --- CHECKS ---
        checks = {
          k3s-integration = import ./tests/k3s-test.nix {
            inherit system pkgs;
            packages = k8sPackages;
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
            envoy
            gettext
            golangci-lint
            govulncheck
          ];

          shellHook = ''
            echo "🛠  Microservices Dev Environment"
            echo "🌍 Gateway Entry Point: http://localhost:8080"
            echo "📊 OTEL Collector (local): http://localhost:4318"
            echo "📊 OTEL Collector (k8s): http://192.168.3.18:8080/v1/traces"
          '';
        };
      }
    );
}
