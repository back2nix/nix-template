{ pkgs, gomod2nix, name, srcBackend, srcFrontend, port, yarnHash }:

let
  # Backend (Go)
  backend = pkgs.buildGoApplication {
    pname = "${name}-backend";
    version = "0.1.0";
    src = srcBackend;
    modules = srcBackend + "/gomod2nix.toml";

    buildPhase = ''
      go build -o backend cmd/server/main.go
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp backend $out/bin/${name}-backend
    '';
  };

  # Frontend (Vue.js)
  frontend = pkgs.mkYarnPackage {
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
      yarn --offline build
    '';

    installPhase = ''
      mkdir -p $out/static
      cp -r dist/* $out/static/
    '';

    distPhase = "true";
  };

in pkgs.symlinkJoin {
  name = "${name}-service";
  paths = [ backend frontend ];

  postBuild = ''
    mkdir -p $out/bin
    cat > $out/bin/start-${name} <<EOF
#!/bin/sh
export SERVER_STATIC_DIR=${frontend}/static
export HTTP_PORT=${port}
exec ${backend}/bin/${name}-backend
EOF
    chmod +x $out/bin/start-${name}
  '';
}
