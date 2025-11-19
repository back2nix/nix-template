{ pkgs, lib, buildGoApplication ? pkgs.buildGoApplication }:

buildGoApplication {
  pname = "web-backend";
  version = "0.0.1";
  src = ./.;
  pwd = ./.;

  modules = ./gomod2nix.toml;

  ldflags = [
    "-s" "-w"
    "-X main.Version=0.0.1"
  ];

  doCheck = false;

  # --- ДОБАВИТЬ ЭТОТ БЛОК ---
  # Go собирает бинарник с именем модуля ("my-go-app"),
  # переименуем его в "web-backend" для единообразия.
  postInstall = ''
    mv $out/bin/my-go-app $out/bin/web-backend
  '';
  # --------------------------

  meta = with lib; {
    description = "Go gRPC Service + Vue";
    mainProgram = "web-backend";
  };
}
