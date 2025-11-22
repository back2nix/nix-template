{
  description = "Python project with uv";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    nixpkgs-old.url = "github:NixOS/nixpkgs/nixos-23.11";  # Старая версия для Python 3.9
    pyproject-nix = {
      url = "github:pyproject-nix/pyproject.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    uv2nix = {
      url = "github:pyproject-nix/uv2nix";
      inputs.pyproject-nix.follows = "pyproject-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    pyproject-build-systems = {
      url = "github:pyproject-nix/build-system-pkgs";
      inputs.pyproject-nix.follows = "pyproject-nix";
      inputs.uv2nix.follows = "uv2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };
  outputs = { self, nixpkgs, nixpkgs-old, uv2nix, pyproject-nix, pyproject-build-systems }:
    let
      inherit (nixpkgs) lib;
      forAllSystems = lib.genAttrs [
        "x86_64-linux"
      ];
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          pkgs-old = nixpkgs-old.legacyPackages.${system};
          python = pkgs-old.python39;  # Python 3.9 из старого nixpkgs
          workspace = uv2nix.lib.workspace.loadWorkspace { workspaceRoot = ./.; };
          overlay = workspace.mkPyprojectOverlay {
            sourcePreference = "wheel";
          };
          pythonSet = (pkgs.callPackage pyproject-nix.build.packages {
            inherit python;
          }).overrideScope (
            lib.composeManyExtensions [
              pyproject-build-systems.overlays.default
              overlay
            ]
          );
        in
        {
          default = pythonSet.mkVirtualEnv "my-python-project-env" workspace.deps.default;
        }
      );
      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/my-python-project";
        };
      });
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          pkgs-old = nixpkgs-old.legacyPackages.${system};
          python = pkgs-old.python39;
          libs = with pkgs; [
            stdenv.cc.cc.lib
            zlib
            glib
          ];
        in
        {
          default = pkgs.mkShell {
            packages = [
              python
              pkgs.uv
              pkgs.just
            ];
            env = {
              UV_PYTHON = "${python}/bin/python";
              LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath libs;
            };
            shellHook = ''
              echo "🐍 Python Dev Environment (uv)"
              echo "Python $(python --version)"
              echo "uv: $(uv --version)"
              # Автоматически создать и активировать .venv
              if [ ! -d .venv ]; then
                echo "Creating virtual environment..."
                uv venv --python 3.9
              fi
              echo "Activating virtual environment..."
              source .venv/bin/activate
              # Синхронизировать зависимости
              uv sync
            '';
          };
        }
      );
    };
}
