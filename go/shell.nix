{ pkgs, mkGoEnv ? pkgs.mkGoEnv, gomod2nix ? pkgs.gomod2nix }:

let
  goEnv = mkGoEnv { pwd = ./backend; };
in
pkgs.mkShell {
  packages = with pkgs; [
    # --- Go Tools ---
    go
    gomod2nix
    gnumake
    golangci-lint
    gopls

    # --- Node / Yarn ---
    nodejs_20
    yarn          # <--- Теперь используем Yarn

    # --- Protobuf ---
    protobuf
    protoc-gen-go
    protoc-gen-go-grpc
    grpcurl

    # --- Utils ---
    just
    jq
  ];

  shellHook = ''
    echo "🚀 Dev Env: Go + gRPC + Vue (Yarn)"
    echo "Go: $(go version)"
    echo "Yarn: $(yarn --version)"
  '';
}
