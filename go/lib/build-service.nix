{ pkgs, gomod2nix, name, srcBackend, srcFrontend, port }:

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
  frontend = pkgs.stdenv.mkDerivation {
    pname = "${name}-frontend";
    version = "0.1.0";
    src = srcFrontend;

    buildInputs = [ pkgs.nodejs_20 pkgs.yarn ];

    buildPhase = ''
      export HOME=$TMPDIR
      yarn install --frozen-lockfile
      yarn build
    '';

    installPhase = ''
      mkdir -p $out/static
      cp -r dist/* $out/static/
    '';
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
