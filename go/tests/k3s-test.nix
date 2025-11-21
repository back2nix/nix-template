{ system, pkgs, packages }:

let
  baseImageContents = with pkgs; [ busybox cacert gettext ];

  pauseImage = pkgs.dockerTools.streamLayeredImage {
    name = "test.local/pause";
    tag = "local";
    contents = baseImageContents;
    config = { Cmd = [ "/bin/sh" "-c" "sleep inf" ]; };
  };

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
            - { name: APP_ENV, value: "prod" }
            - { name: GREETER_HTTP_PORT, value: "8081" }
            - { name: GREETER_GRPC_PORT, value: "50051" }
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
            - { name: APP_ENV, value: "prod" }
            - { name: SHELL_HTTP_PORT, value: "9002" }
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
            command: ["/bin/start-gateway"]
            env:
            - { name: GATEWAY_HTTP_PORT, value: "8085" }
            - { name: GREETER_HOST, value: "127.0.0.1" }
            - { name: GREETER_PORT, value: "8081" }
  '';

in pkgs.testers.nixosTest {
  name = "microservices-k3s-integration";

  nodes.machine = { config, pkgs, ... }: {
    services.k3s = {
      enable = true;
      role = "server";
      extraFlags = toString [
        "--disable traefik"
        "--disable metrics-server"
        "--disable coredns"
        "--disable local-storage"
        "--pause-image test.local/pause:local"
      ];
    };

    environment.variables.KUBECONFIG = "/etc/rancher/k3s/k3s.yaml";

    environment.systemPackages = with pkgs; [ kubectl jq ];
    networking.firewall.allowedTCPPorts = [ 6443 8080 8081 8085 9002 50051 ];
    virtualisation.memorySize = 2048;
    virtualisation.diskSize = 4096;
  };

  testScript = ''
    start_all()

    machine.wait_for_unit("k3s")
    machine.wait_until_succeeds("test -f /etc/rancher/k3s/k3s.yaml")

    machine.succeed("${pauseImage} | ctr -n k8s.io image import -")
    machine.wait_until_succeeds("kubectl cluster-info")

    machine.succeed("${gatewayImage} | ctr -n k8s.io image import -")
    machine.succeed("${greeterImage} | ctr -n k8s.io image import -")
    machine.succeed("${shellImage} | ctr -n k8s.io image import -")

    machine.succeed("kubectl apply -f ${k8sManifests}")

    # Ждем пока pod'ы создадутся
    machine.wait_until_succeeds("kubectl get pods | grep greeter")
    machine.wait_until_succeeds("kubectl get pods | grep shell")
    machine.wait_until_succeeds("kubectl get pods | grep gateway")

    # Даём время на первый запуск gateway
    import time
    time.sleep(5)

    # DEBUGGING: Получаем имя и логи gateway pod'а
    print("\n========= GATEWAY POD STATUS =========")
    gateway_status = machine.succeed("kubectl get pods | grep gateway")
    print(gateway_status)

    gateway_pod = machine.succeed("kubectl get pods -o name | grep gateway").strip()
    print(f"Gateway pod name: {gateway_pod}")

    print("\n========= GATEWAY POD LOGS =========")
    logs = machine.succeed(f"kubectl logs {gateway_pod} 2>&1 || echo 'No logs yet'")
    print(logs)
    print("====================================\n")

    # Теперь ждём Running статус
    machine.wait_until_succeeds("kubectl get pods | grep gateway | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep greeter | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep shell | grep Running")

    print("Checking Gateway Health on :8085...")
    machine.wait_until_succeeds("curl -sSf http://localhost:8085/health | grep 'ok'", timeout=30)

    print("Checking Gateway -> Greeter proxy on :8085...")
    output = machine.succeed("curl -s 'http://localhost:8085/api/greeter/api/hello?name=NixOS'")

    print(f"\n========= RESPONSE FROM GATEWAY =========\n{output}\n=========================================\n")

    assert "Hello" in output and "NixOS" in output

    print("✅ All integration tests passed!")
  '';
}
