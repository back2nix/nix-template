{ pkgs, app }:

let
  # Собираем минимальное окружение (если нужны сертификаты или tzdata)
  myEnv = pkgs.buildEnv {
    name = "image-root";
    paths = [
      app
      pkgs.cacert    # Корневые сертификаты (для HTTPS запросов)
      pkgs.tzdata    # Временные зоны
      # pkgs.bash    # Можно добавить bash для отладки, но увеличит размер
    ];
    pathsToLink = [ "/bin" "/share" "/etc" ];
  };
in
pkgs.dockerTools.buildImage {
  name = app.pname;
  tag = "latest";

  # Если собираем на Linux для Linux, можно оставить как есть.
  # Если нужно кросс-компилировать, здесь нужны доп. настройки,
  # но для шаблона берем архитектуру хоста или явно указываем "x86_64"
  architecture = pkgs.go.GOARCH;

  copyToRoot = myEnv;

  config = {
    # Запускаем бинарник. Путь зависит от имени пакета в default.nix
    Cmd = [ "/bin/${app.meta.mainProgram}" ];

    ExposedPorts = {
      "8080/tcp" = {};
    };

    Env = [
      "SSL_CERT_FILE=/etc/ssl/certs/ca-bundle.crt" # Важно для HTTPS
    ];
  };
}
