{
  description = "Python project with uv";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    nixpkgs-old.url = "github:NixOS/nixpkgs/nixos-23.11";
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
      forAllSystems = lib.genAttrs ["x86_64-linux"];
    in
    {
      packages = forAllSystems (system:
        let
          # ДОБАВЬ allowUnfree
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
          pkgs-old = nixpkgs-old.legacyPackages.${system};
          python = pkgs-old.python39;

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
          # ДОБАВЬ allowUnfree
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
          pkgs-old = nixpkgs-old.legacyPackages.${system};
          python = pkgs-old.python39;

          # ДОБАВЬ CUDA библиотеки
          cudaLibs = with pkgs.cudaPackages; [
            cudatoolkit
            cuda_cudart
            cuda_nvrtc
            libcublas
            libcufft
            libcurand
            libcusparse
            libcusolver
            cudnn
          ];

          libs = with pkgs; [
            stdenv.cc.cc.lib
            zlib
            glib
          ] ++ cudaLibs;
        in
        {
          default = pkgs.mkShell {
            packages = [
              python
              pkgs.uv
              pkgs.just
              pkgs.aria2
            ];
            env = {
              UV_PYTHON = "${python}/bin/python";
              # ВАЖНО: добавь /run/opengl-driver/lib для NVIDIA драйвера
              LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath libs + ":/run/opengl-driver/lib";
              CUDA_HOME = "${pkgs.cudaPackages.cudatoolkit}";
              CUDA_PATH = "${pkgs.cudaPackages.cuda_cudart}";
            };
            shellHook = ''
              echo "🐍 Python Dev Environment (uv)"
              echo "uv: $(uv --version)"

              if [ ! -d .venv ]; then
                echo "Creating virtual environment..."
                uv venv
              fi

              echo "Activating virtual environment..."
              source .venv/bin/activate

              echo "Installing dependencies with CUDA support..."
              uv sync --extra cuda

              echo "Checking CUDA availability..."
              python -c "import torch; print(f'✓ PyTorch: {torch.__version__}'); print(f'✓ CUDA available: {torch.cuda.is_available()}'); print(f'✓ Device: {torch.cuda.get_device_name(0) if torch.cuda.is_available() else \"N/A\"}')" 2>/dev/null || echo "⚠ CUDA check failed"
            '';
          };
        }
      );
    };
}
