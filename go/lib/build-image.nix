{ pkgs, servicePkg, name, configEnv ? [] }:

pkgs.dockerTools.buildLayeredImage {
  name = name;
  tag = "latest";
  created = "now";

  contents = [
    servicePkg
    pkgs.cacert
    pkgs.tzdata
    pkgs.bash
    pkgs.coreutils
  ];

  config = {
    # В K8s мы будем переопределять CMD/Entrypoint, но база должна быть
    Cmd = [ "/bin/start-${name}" ];
    Env = [
      "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
      "TZ=UTC"
    ] ++ configEnv;

    WorkingDir = "/";
  };
}
