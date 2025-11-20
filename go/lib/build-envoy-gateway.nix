{ pkgs, name, port }:

let
  # Копируем конфиг в Store
  envoyConfig = ../services/gateway/envoy.tmpl.yaml;

  # Скрипт запуска, который подменяет переменные окружения в конфиге
  startScript = pkgs.writeShellScriptBin "start-${name}" ''
    set -e

    # Значения по умолчанию
    export GATEWAY_HTTP_PORT=''${GATEWAY_HTTP_PORT:-${port}}
    export GREETER_HOST=''${GREETER_HOST:-localhost}
    export GREETER_PORT=''${GREETER_PORT:-8081}

    echo "🚀 Starting Envoy Gateway..."
    echo "   Port: $GATEWAY_HTTP_PORT"
    echo "   Upstream Greeter: $GREETER_HOST:$GREETER_PORT"

    # Создаем временный конфиг с подставленными значениями
    # Используем envsubst из пакета gettext
    ${pkgs.gettext}/bin/envsubst < ${envoyConfig} > ./envoy.yaml

    # Запускаем Envoy
    exec ${pkgs.envoy}/bin/envoy -c ./envoy.yaml --service-cluster ${name} --service-node ${name}
  '';

in pkgs.symlinkJoin {
  name = "${name}-service";
  paths = [ startScript pkgs.envoy ];
}
