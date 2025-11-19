{ pkgs, lib }:

let
  # ЗАМЕНИ ЭТОТ ХЕШ ПОСЛЕ ПЕРВОГО ЗАПУСКА (Nix сам подскажет правильный)
  yarnDepsHash = "sha256-h7PdfPxL6rpTgaoTPmCDvxhQZwD4u9shYapxAt2NX7A=";
in
pkgs.mkYarnPackage rec {
  pname = "web-frontend";
  version = "0.0.1";
  src = ./.;

  # Это позволяет Nix скачать пакеты yarn заранее
  offlineCache = pkgs.fetchYarnDeps {
    yarnLock = src + "/yarn.lock";
    sha256 = yarnDepsHash;
  };

  # Настройка сборки
  configurePhase = ''
    cp -r $node_modules node_modules
    chmod -R +w node_modules
  '';

  buildPhase = ''
    export HOME=$(mktemp -d)
    yarn --offline build
  '';

  installPhase = ''
    mkdir -p $out
    # Копируем результат сборки (dist)
    cp -r dist $out/dist
  '';

  # Отключаем лишние фазы mkYarnPackage
  distPhase = "true";
  doCheck = false;
}
