{ pkgs, mkGoEnv ? pkgs.mkGoEnv, gomod2nix ? pkgs.gomod2nix }:

let
  goEnv = mkGoEnv { pwd = ./.; };
in
pkgs.mkShell {
  packages = with pkgs; [
    # Основные инструменты
    go
    gomod2nix
    gnumake

    # Линтеры и инструменты
    golangci-lint
    gopls
    gotools
    delve
    go-tools

    # Утилиты
    just
    jq
    grpcurl # Удобно для ручного тестирования gRPC

    # Protobuf инструменты
    protobuf             # Компилятор protoc
    protoc-gen-go        # Генерация структур Go
    protoc-gen-go-grpc   # Генерация gRPC сервиса
  ];

  shellHook = ''
    echo "🚀 Go gRPC Dev Environment Loaded"
    echo "Go version: $(go version)"
    echo "Protoc version: $(protoc --version)"
  '';
}
