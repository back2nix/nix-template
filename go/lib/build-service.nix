{ pkgs, gomod2nix, name, srcBackend, srcFrontend ? null, port, yarnHash ? null, basePath ? "/", otelEndpoint ? "http://localhost:4318/v1/traces", gatewayUrl ? "http://localhost:8080" }:

let
  # 1. Объединяем backend и pkg
  combinedSrc = pkgs.runCommand "combined-src" {} ''
    mkdir -p $out

    # Копируем backend
    cp -r ${srcBackend}/* $out/

    # Копируем pkg
    mkdir -p $out/pkg
    cp -r ${../pkg}/* $out/pkg/

    # ВАЖНО: Удаляем go.mod/go.sum из pkg, чтобы он стал частью модуля сервиса
    # и не требовал replace директив
    rm -f $out/pkg/go.mod $out/pkg/go.sum
  '';

  # 2. Сборка Backend (Go)
  backend = pkgs.buildGoApplication {
    pname = "${name}-backend";
    version = "0.1.0";
    src = combinedSrc;
    modules = srcBackend + "/gomod2nix.toml";

    CGO_ENABLED = 0;

    buildPhase = ''
      go build -o backend cmd/server/main.go
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp backend $out/bin/${name}-backend
    '';
  };

  # 3. Сборка Frontend (Vue.js) - Опционально
  hasFrontend = srcFrontend != null;

  frontend = if hasFrontend then pkgs.mkYarnPackage {
    pname = "${name}-frontend";
    version = "0.1.0";
    src = srcFrontend;

    offlineCache = pkgs.fetchYarnDeps {
      yarnLock = srcFrontend + "/yarn.lock";
      hash = yarnHash;
    };

    configurePhase = ''
      export HOME=$(mktemp -d)
      cp -r $node_modules node_modules
      chmod +w node_modules
    '';

    buildPhase = ''
      export VITE_BASE_PATH="${basePath}"
      export VITE_OTEL_ENDPOINT="${otelEndpoint}"
      export VITE_GATEWAY_URL="${gatewayUrl}"
      # Для shell дополнительно передаем remote URLs
      ${if name == "shell" then ''
        export VITE_LANDING_REMOTE_URL="${gatewayUrl}/api/landing/remoteEntry.js"
        export VITE_CHAT_REMOTE_URL="${gatewayUrl}/api/chat/remoteEntry.js"
      '' else ""}
      yarn --offline build
    '';

    installPhase = ''
      mkdir -p $out/static
      cp -r dist/* $out/static/
    '';

    distPhase = "true";
  } else null;

  # Формируем список путей для объединения
  servicePaths = [ backend ] ++ (if hasFrontend then [ frontend ] else []);

  # Используем абсолютный путь к статике из Nix store
  staticPath = if hasFrontend then "${frontend}/static" else "";

  # Uppercase имя сервиса для префикса переменных
  servicePrefix = pkgs.lib.toUpper name;

  # Формируем скрипт запуска с правильными префиксами
  startScript = ''
#!/bin/sh
export ${servicePrefix}_SERVER_HTTP_PORT=${port}
${if hasFrontend then "export ${servicePrefix}_SERVER_STATIC_DIR=${staticPath}" else ""}
exec ${backend}/bin/${name}-backend
'';

in pkgs.symlinkJoin {
  name = "${name}-service";
  paths = servicePaths;

  postBuild = ''
    mkdir -p $out/bin
    cat > $out/bin/start-${name} <<EOF
${startScript}
EOF
    chmod +x $out/bin/start-${name}
  '';
}
