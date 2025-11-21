{ pkgs, name, port }:

let
  envoyConfig = ../services/gateway/envoy.tmpl.yaml;

  startScript = pkgs.writeShellScriptBin "start-${name}" ''
    set -e

    # 1. Настройка порта Gateway
    export GATEWAY_HTTP_PORT=''${GATEWAY_HTTP_PORT:-${port}}

    # 2. Настройка Greeter Service
    # Если GREETER_HOST не задан явно, пытаемся взять из K8s Env Var (GREETER_SERVICE_HOST)
    if [ -z "$GREETER_HOST" ]; then
      if [ -n "$GREETER_SERVICE_HOST" ]; then
        echo "Using K8s Service Discovery for Greeter..."
        export GREETER_HOST="$GREETER_SERVICE_HOST"
        # Пытаемся найти HTTP порт. Если сервисный порт один, он в GREETER_SERVICE_PORT.
        # Если их несколько (как у нас: 50051 и 8081), K8s создаст vars по именам портов, но это сложно.
        # Для надежности в тесте мы используем дефолт 8081, так как ClusterIP порт совпадает с ContainerPort.
        export GREETER_PORT="8081"
      else
        export GREETER_HOST="127.0.0.1"
        export GREETER_PORT="8081"
      fi
    else
      export GREETER_PORT=''${GREETER_PORT:-8081}
    fi

    # 3. Настройка Shell Service
    if [ -z "$SHELL_HOST" ]; then
      if [ -n "$SHELL_SERVICE_HOST" ]; then
        echo "Using K8s Service Discovery for Shell..."
        export SHELL_HOST="$SHELL_SERVICE_HOST"
        export SHELL_PORT="$SHELL_SERVICE_PORT"
      else
        export SHELL_HOST="127.0.0.1"
        export SHELL_PORT="9002"
      fi
    else
      export SHELL_PORT=''${SHELL_PORT:-9002}
    fi

    # 4. Настройка OTel Collector
    # Важно: если DNS нет, имя "otel-collector" сломает Envoy. Используем IP по умолчанию.
    export OTEL_COLLECTOR_HOST=''${OTEL_COLLECTOR_HOST:-127.0.0.1}
    export OTEL_COLLECTOR_PORT=''${OTEL_COLLECTOR_PORT:-4317}

    echo "🚀 Starting Envoy Gateway..."
    echo "   Port: $GATEWAY_HTTP_PORT"
    echo "   Upstream Greeter: $GREETER_HOST:$GREETER_PORT"
    echo "   Upstream Shell:   $SHELL_HOST:$SHELL_PORT"
    echo "   OTel Collector:   $OTEL_COLLECTOR_HOST:$OTEL_COLLECTOR_PORT"

    mkdir -p /tmp

    ENVOY_CONFIG_PATH="/tmp/envoy-${name}.yaml"
    ${pkgs.gettext}/bin/envsubst < ${envoyConfig} > "$ENVOY_CONFIG_PATH"

    echo "   Config generated at: $ENVOY_CONFIG_PATH"

    exec ${pkgs.envoy}/bin/envoy -c "$ENVOY_CONFIG_PATH" --service-cluster ${name} --service-node ${name}
  '';

in pkgs.symlinkJoin {
  name = "${name}-service";
  paths = [ startScript pkgs.envoy ];
}
