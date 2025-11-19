{
  description = "Python project with uv";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true; # Нужно для CUDA, если понадобится
        };

        # Библиотеки, которые нужны python-пакетам (через LD_LIBRARY_PATH)
        # Сюда добавляем libsndfile, cuda, ffmpeg если нужно
        libs = with pkgs; [
          stdenv.cc.cc.lib
          zlib
          glib
          # libsndfile  # Для аудио
          # ffmpeg      # Для обработки медиа
        ];
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Основные инструменты
            python311
            uv
            just

            # Системные зависимости (если нужны хедеры при сборке)
            # pkg-config
          ];

          # Настройка переменных окружения
          env = {
            # Заставляем uv использовать python из nix store,
            # чтобы не качал свой toolchain, который может конфликтовать с glibc
            UV_PYTHON = "${pkgs.python311}/bin/python";

            # Указываем путь к библиотекам для динамической линковки
            LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath libs;
          };

          shellHook = ''
            echo "🐍 Python Dev Environment (uv)"
            echo "Python: $(python --version)"
            echo "uv: $(uv --version)"
          '';
        };
      }
    );
}
