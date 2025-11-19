{ pkgs, gomod2nix, name, srcBackend, port }:

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

in pkgs.symlinkJoin {
  name = "${name}-service";
  paths = [ backend ];

  postBuild = ''
    mkdir -p $out/bin
    cat > $out/bin/start-${name} <<EOF
#!/bin/sh
export HTTP_PORT=${port}
exec ${backend}/bin/${name}-backend
EOF
    chmod +x $out/bin/start-${name}
  '';
}
