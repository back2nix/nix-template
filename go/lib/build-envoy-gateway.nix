{ pkgs, name, port }:

let
  envoyConfig = ../services/gateway/envoy.tmpl.yaml;

  startScript = pkgs.writeShellScriptBin "start-${name}" ''
    set -e

    # 1. Настройка порта Gateway
    export GATEWAY_HTTP_PORT=''${GATEWAY_HTTP_PORT:-${port}}

    # 2. Настройка Landing Service
    if [ -z "$LANDING_HOST" ]; then
      if [ -n "$LANDING_SERVICE_HOST" ]; then
        echo "Using K8s Service Discovery for Landing..."
        export LANDING_HOST="$LANDING_SERVICE_HOST"
        export LANDING_PORT="8081"
      else
        export LANDING_HOST="127.0.0.1"
        export LANDING_PORT="8081"
      fi
    else
      export LANDING_PORT=''${LANDING_PORT:-8081}
    fi

    # 3. Настройка Chat Service
    if [ -z "$CHAT_HOST" ]; then
      if [ -n "$CHAT_SERVICE_HOST" ]; then
        echo "Using K8s Service Discovery for Chat..."
        export CHAT_HOST="$CHAT_SERVICE_HOST"
        export CHAT_PORT="8082"
      else
        export CHAT_HOST="127.0.0.1"
        export CHAT_PORT="8082"
      fi
    else
      export CHAT_PORT=''${CHAT_PORT:-8082}
    fi

    # 4. Настройка Notification Service
    if [ -z "$NOTIFICATION_HOST" ]; then
      if [ -n "$NOTIFICATION_SERVICE_HOST" ]; then
        echo "Using K8s Service Discovery for Notification..."
        export NOTIFICATION_HOST="$NOTIFICATION_SERVICE_HOST"
        export NOTIFICATION_PORT="8085"
      else
        export NOTIFICATION_HOST="127.0.0.1"
        export NOTIFICATION_PORT="8085"
      fi
    else
      export NOTIFICATION_PORT=''${NOTIFICATION_PORT:-8085}
    fi

    # 5. Настройка Shell Service
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

    # 6. Настройка OTel Collector
    export OTEL_COLLECTOR_HOST=''${OTEL_COLLECTOR_HOST:-127.0.0.1}
    export OTEL_COLLECTOR_PORT=''${OTEL_COLLECTOR_PORT:-4317}

    echo "🚀 Starting Envoy Gateway..."
    echo "   Port: $GATEWAY_HTTP_PORT"
    echo "   Upstream Landing:      $LANDING_HOST:$LANDING_PORT"
    echo "   Upstream Chat:         $CHAT_HOST:$CHAT_PORT"
    echo "   Upstream Notification: $NOTIFICATION_HOST:$NOTIFICATION_PORT"
    echo "   Upstream Shell:        $SHELL_HOST:$SHELL_PORT"
    echo "   OTel Collector:        $OTEL_COLLECTOR_HOST:$OTEL_COLLECTOR_PORT"

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
