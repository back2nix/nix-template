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

    # Создаем /tmp если его нет
    mkdir -p /tmp

    # Создаем временный конфиг с подставленными значениями в /tmp
    ENVOY_CONFIG_PATH="/tmp/envoy-${name}.yaml"
    ${pkgs.gettext}/bin/envsubst < ${envoyConfig} > "$ENVOY_CONFIG_PATH"

    echo "   Config generated at: $ENVOY_CONFIG_PATH"

    # Запускаем Envoy
    exec ${pkgs.envoy}/bin/envoy -c "$ENVOY_CONFIG_PATH" --service-cluster ${name} --service-node ${name}
  '';

in pkgs.symlinkJoin {
  name = "${name}-service";
  paths = [ startScript pkgs.envoy ];
}
