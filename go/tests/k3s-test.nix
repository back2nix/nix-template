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
    # --- GREETER SERVICE ---
    apiVersion: v1
    kind: Service
    metadata: { name: greeter }
    spec:
      selector: { app: greeter }
      ports:
        - name: grpc
          port: 50051
          targetPort: 50051
        - name: http
          port: 8081
          targetPort: 8081
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata: { name: greeter }
    spec:
      selector: { matchLabels: { app: greeter } }
      template:
        metadata: { labels: { app: greeter } }
        spec:
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
    # --- SHELL SERVICE ---
    apiVersion: v1
    kind: Service
    metadata: { name: shell }
    spec:
      selector: { app: shell }
      ports:
        - name: http
          port: 9002
          targetPort: 9002
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata: { name: shell }
    spec:
      selector: { matchLabels: { app: shell } }
      template:
        metadata: { labels: { app: shell } }
        spec:
          containers:
          - name: shell
            image: shell:latest
            imagePullPolicy: Never
            command: ["/bin/shell-backend"]
            env:
            - { name: APP_ENV, value: "prod" }
            - { name: SHELL_HTTP_PORT, value: "9002" }

    ---
    # --- GATEWAY SERVICE ---
    apiVersion: v1
    kind: Service
    metadata: { name: gateway }
    spec:
      type: NodePort
      selector: { app: gateway }
      ports:
        - name: http
          port: 8085
          targetPort: 8085
          nodePort: 30085
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata: { name: gateway }
    spec:
      selector: { matchLabels: { app: gateway } }
      template:
        metadata: { labels: { app: gateway } }
        spec:
          containers:
          - name: gateway
            image: gateway:latest
            imagePullPolicy: Never
            command: ["/bin/start-gateway"]
            env:
            - { name: GATEWAY_HTTP_PORT, value: "8085" }
            # Мы не задаем GREETER_HOST/SHELL_HOST явно.
            # Скрипт start-gateway сам возьмет их из K8s Env Vars (GREETER_SERVICE_HOST),
            # которые K8s пробросит, так как сервисы созданы раньше деплоймента.
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
        "--disable coredns" # CoreDNS отключен, так как нет образа
        "--disable local-storage"
        "--pause-image test.local/pause:local"
      ];
    };

    environment.variables.KUBECONFIG = "/etc/rancher/k3s/k3s.yaml";

    environment.systemPackages = with pkgs; [ kubectl jq netcat ];
    networking.firewall.allowedTCPPorts = [ 6443 30085 ];
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

    machine.wait_until_succeeds("kubectl get pods | grep greeter")
    machine.wait_until_succeeds("kubectl get pods | grep shell")
    machine.wait_until_succeeds("kubectl get pods | grep gateway")

    machine.wait_until_succeeds("kubectl get pods | grep gateway | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep greeter | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep shell | grep Running")

    print("Waiting for Gateway Service on port 30085...")
    machine.sleep(10)
    machine.wait_until_succeeds("nc -z localhost 30085")

    print("Checking Gateway Health on :30085...")
    machine.wait_until_succeeds("curl -sSf http://localhost:30085/health | grep 'ok'", timeout=60)

    print("Checking Gateway -> Greeter proxy on :30085...")
    output = machine.succeed("curl -v -s 'http://localhost:30085/api/greeter/api/hello?name=NixOS'")

    print(f"\n========= RESPONSE FROM GATEWAY =========\n{output}\n=========================================\n")

    assert "Hello" in output and "NixOS" in output

    print("✅ All integration tests passed!")
  '';
}
