{ pkgs, lib, buildGoApplication ? pkgs.buildGoApplication }:

let
  # Читаем версию из git тега или ставим dev, если нет git
  version = "0.0.1";
in
buildGoApplication {
  pname = "my-go-app";
  inherit version;

  # Путь к корню исходников
  src = ./.;
  pwd = ./.; # Важно для gomod2nix

  # Это тот самый важный момент, чтобы не считать хеши вручную.
  # Файл gomod2nix.toml генерируется командой `gomod2nix` (см. инструкцию ниже)
  modules = ./gomod2nix.toml;

  # Флаги компиляции (убираем отладочную инфу + вшиваем версию)
  ldflags = [
    "-s" "-w"
    "-X main.Version=${version}"
  ];

  # Если нужно запускать тесты при сборке - true.
  # Часто выключают для ускорения CI, если тесты гоняются отдельным шагом.
  doCheck = false;

  meta = with lib; {
    description = "A generic Go application";
    license = licenses.mit;
    mainProgram = "my-go-app"; # Имя бинарника
  };
}
