{ system, pkgs, packages }:

let
  # 1. Базовый набор утилит для контейнеров
  baseImageContents = with pkgs; [ busybox cacert ];

  # 2. Заглушка pause-образа (нужна для K3s)
  pauseImage = pkgs.dockerTools.streamLayeredImage {
    name = "test.local/pause";
    tag = "local";
    contents = baseImageContents;
    config = { Cmd = [ "/bin/sh" "-c" "sleep inf" ]; };
  };

  # 3. Образы приложений
  gatewayImage = pkgs.dockerTools.streamLayeredImage {
    name = "gateway";
    tag = "latest";
    contents = baseImageContents ++ [ packages.gateway ];
  };

  greeterImage = pkgs.dockerTools.streamLayeredImage {
    name = "greeter";
    tag = "latest";
    contents = baseImageContents ++ [ packages.greeter ];
  };

  shellImage = pkgs.dockerTools.streamLayeredImage {
    name = "shell";
    tag = "latest";
    contents = baseImageContents ++ [ packages.shell ];
  };

  # 4. Kubernetes манифесты
  # Используем hostNetwork: true, чтобы избежать проблем с CNI/DNS в песочнице
  # Используем command: [...] чтобы переопределить entrypoint и задать порты через ENV
  k8sManifests = pkgs.writeText "app-deployment.yaml" ''
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata: { name: greeter }
    spec:
      selector: { matchLabels: { app: greeter } }
      template:
        metadata: { labels: { app: greeter } }
        spec:
          hostNetwork: true
          containers:
          - name: greeter
            image: greeter:latest
            imagePullPolicy: Never
            command: ["/bin/greeter-backend"]
            env:
            - { name: HTTP_PORT, value: "8081" }
            - { name: GRPC_PORT, value: "50051" }
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata: { name: shell }
    spec:
      selector: { matchLabels: { app: shell } }
      template:
        metadata: { labels: { app: shell } }
        spec:
          hostNetwork: true
          containers:
          - name: shell
            image: shell:latest
            imagePullPolicy: Never
            command: ["/bin/shell-backend"]
            env:
            - { name: HTTP_PORT, value: "9002" }
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata: { name: gateway }
    spec:
      selector: { matchLabels: { app: gateway } }
      template:
        metadata: { labels: { app: gateway } }
        spec:
          hostNetwork: true
          containers:
          - name: gateway
            image: gateway:latest
            imagePullPolicy: Never
            command: ["/bin/gateway-backend"]
            env:
            - { name: HTTP_PORT, value: "8080" }
            - { name: GREETER_URL, value: "http://127.0.0.1:8081" }
  '';

in pkgs.testers.nixosTest {
  name = "microservices-k3s-integration";

  nodes.machine = { config, pkgs, ... }: {
    # Настройка K3s сервера
    services.k3s = {
      enable = true;
      role = "server";
      extraFlags = toString [
        "--disable traefik"
        "--disable metrics-server"
        "--disable coredns"        # Не нужен при hostNetwork
        "--disable local-storage"  # Не нужен без PVC
        "--pause-image test.local/pause:local"
      ];
    };

    # Утилиты и открытые порты
    environment.systemPackages = with pkgs; [ kubectl jq ];
    networking.firewall.allowedTCPPorts = [ 6443 8080 8081 9002 50051 ];

    # Ресурсы VM
    virtualisation.memorySize = 2048;
    virtualisation.diskSize = 4096;
  };

  testScript = ''
    start_all()

    # 1. Инициализация K3s и загрузка Pause образа
    machine.wait_for_unit("k3s")
    machine.succeed("${pauseImage} | ctr -n k8s.io image import -")
    machine.wait_until_succeeds("kubectl cluster-info")

    # 2. Загрузка образов приложений
    machine.succeed("${gatewayImage} | ctr -n k8s.io image import -")
    machine.succeed("${greeterImage} | ctr -n k8s.io image import -")
    machine.succeed("${shellImage} | ctr -n k8s.io image import -")

    # 3. Применение манифестов
    machine.succeed("kubectl apply -f ${k8sManifests}")

    # 4. Ожидание готовности подов
    machine.wait_until_succeeds("kubectl get pods | grep gateway | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep greeter | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep shell | grep Running")

    # 5. Проверка Health Check Gateway
    print("Checking Gateway Health...")
    machine.succeed("curl -sSf http://localhost:8080/health | grep 'ok'")

    # 6. Проверка сквозного запроса (Proxy)
    print("Checking Gateway -> Greeter proxy...")

    # Выполняем запрос и сохраняем вывод
    # Используем -s (silent), чтобы curl не спамил прогрессом
    output = machine.succeed("curl -s 'http://localhost:8080/api/greeter/api/hello?name=NixOS'")

    # Выводим ответ в лог для отладки
    print(f"\n========= RESPONSE FROM GATEWAY =========\n{output}\n=========================================\n")

    # Проверяем содержимое ответа
    assert "Hello" in output and "NixOS" in output

    print("✅ All integration tests passed!")
  '';
}
