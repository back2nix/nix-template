{
  description = "My collection of Nix flakes templates";

  outputs = { self }: {

    templates = {

      # Шаблон Go
      go = {
        path = ./go;
        description = "Production-ready Go template with gomod2nix and Docker support";
        welcomeText = ''
          # Go Flake Template initialized!
          1. Update `go.mod`: go mod edit -module my-project
          2. Generate config: nix develop --command gomod2nix
          3. Run: nix run
        '';
      };

      # Шаблон Python + uv
      python = {
        path = ./python;
        description = "Python template using 'uv' for dependency management and Nix for system libs";
        welcomeText = ''
          # Python + uv Flake Template initialized!

          1. Review `pyproject.toml` (name, dependencies).
          2. Install dependencies:
             just setup
          3. Run project:
             just run
        '';
      };

    };

    templates.default = self.templates.go;
  };
}
