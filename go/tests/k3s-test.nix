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

  landingImage = pkgs.dockerTools.streamLayeredImage {
    name = "landing";
    tag = "latest";
    contents = baseImageContents ++ [ packages.landing ];
  };

  shellImage = pkgs.dockerTools.streamLayeredImage {
    name = "shell";
    tag = "latest";
    contents = baseImageContents ++ [ packages.shell ];
  };

  k8sManifests = pkgs.writeText "app-deployment.yaml" ''
    ---
    # --- LANDING SERVICE ---
    apiVersion: v1
    kind: Service
    metadata: { name: landing }
    spec:
      selector: { app: landing }
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
    metadata: { name: landing }
    spec:
      selector: { matchLabels: { app: landing } }
      template:
        metadata: { labels: { app: landing } }
        spec:
          containers:
          - name: landing
            image: landing:latest
            imagePullPolicy: Never
            command: ["/bin/landing-backend"]
            env:
            - { name: APP_ENV, value: "prod" }
            - { name: LANDING_HTTP_PORT, value: "8081" }
            - { name: LANDING_GRPC_PORT, value: "50051" }

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
            # Env variables for service discovery are injected by K8s
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
    machine.succeed("${landingImage} | ctr -n k8s.io image import -")
    machine.succeed("${shellImage} | ctr -n k8s.io image import -")

    machine.succeed("kubectl apply -f ${k8sManifests}")

    machine.wait_until_succeeds("kubectl get pods | grep landing")
    machine.wait_until_succeeds("kubectl get pods | grep shell")
    machine.wait_until_succeeds("kubectl get pods | grep gateway")

    machine.wait_until_succeeds("kubectl get pods | grep gateway | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep landing | grep Running")
    machine.wait_until_succeeds("kubectl get pods | grep shell | grep Running")

    print("Waiting for Gateway Service on port 30085...")
    machine.sleep(10)
    machine.wait_until_succeeds("nc -z localhost 30085")

    print("Checking Gateway Health on :30085...")
    machine.wait_until_succeeds("curl -sSf http://localhost:30085/health | grep 'ok'", timeout=60)

    print("Checking Gateway -> Landing proxy on :30085...")
    # Envoy rewrite: /api/landing/hello -> /hello
    # Backend handler: /hello
    output = machine.succeed("curl -v -s 'http://localhost:30085/api/landing/hello?name=NixOS'")

    print(f"\n========= RESPONSE FROM GATEWAY =========\n{output}\n=========================================\n")

    assert "Hello" in output and "NixOS" in output

    print("✅ All integration tests passed!")
  '';
}
