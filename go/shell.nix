{ pkgs, mkGoEnv ? pkgs.mkGoEnv, gomod2nix ? pkgs.gomod2nix }:

let
  # Создаем окружение, которое понимает Go IDE
  goEnv = mkGoEnv { pwd = ./.; };
in
pkgs.mkShell {
  packages = with pkgs; [
    # Основные инструменты
    go
    gomod2nix # Утилита для генерации gomod2nix.toml
    gnumake

    # Линтеры и инструменты разработки
    golangci-lint
    gopls       # LSP сервер
    gotools     # goimports, etc.
    delve       # Отладчик
    go-tools    # staticcheck и прочее

    # Полезные утилиты
    just        # Альтернатива Make
    jq
  ];

  # Переменные окружения для shell
  shellHook = ''
    echo "🚀 Go Dev Environment Loaded"
    echo "Go version: $(go version)"

    # Настройка pre-commit хуков, если нужно
    # git config core.hooksPath .git-hooks
  '';
}
