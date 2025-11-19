{ pkgs, lib, backend, frontend ? null }:

pkgs.stdenv.mkDerivation {
  name = "full-app";
  phases = [ "installPhase" ];
  buildInputs = [ pkgs.makeWrapper ];

  installPhase = ''
    mkdir -p $out/bin $out/share/web

    # 1. Копируем Backend (используем install для установки прав rwx)
    # Было: cp ${backend}/bin/* $out/bin/
    install -Dm755 ${backend}/bin/${backend.meta.mainProgram} $out/bin/${backend.meta.mainProgram}

    # 2. Копируем Frontend (если есть)
    ${lib.optionalString (frontend != null) ''
      echo "Copying frontend dist..."
      cp -r ${frontend}/dist $out/share/web/static
    ''}

    # 3. Делаем обертку
    makeWrapper $out/bin/${backend.meta.mainProgram} $out/bin/app-wrapped \
      --set SERVER_STATIC_DIR $out/share/web/static
  '';
}
