{
  description = "My collection of Nix flakes templates";

  outputs = { self }: {

    templates = {

      # Объявляем шаблон "go"
      go = {
        path = ./go;
        description = "Production-ready Go template with gomod2nix and Docker support";

        # Это сообщение увидит пользователь сразу после создания проекта
        welcomeText = ''
          # Go Flake Template initialized!

          1. Update `go.mod` with your module name:
             go mod edit -module my-new-project

          2. Generate `gomod2nix.toml` (REQUIRED):
             nix develop --command gomod2nix

          3. Run the project:
             nix run
        '';
      };

      # В будущем добавите сюда другие:
      # python = { path = ./python; description = "..."; };

    };

    # Шаблон по умолчанию (если не указать имя при создании)
    templates.default = self.templates.go;
  };
}
