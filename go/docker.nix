{ pkgs, app }:

let
  myEnv = pkgs.buildEnv {
    name = "image-root";
    paths = [
      app
      pkgs.cacert
      pkgs.tzdata
    ];
    pathsToLink = [ "/bin" "/share" "/etc" ];
  };
in
pkgs.dockerTools.buildImage {
  name = app.pname;
  tag = "latest";
  architecture = pkgs.go.GOARCH;

  copyToRoot = myEnv;

  config = {
    Cmd = [ "/bin/${app.meta.mainProgram}" ];

    ExposedPorts = {
      "8080/tcp" = {};  # HTTP (если будет)
      "50051/tcp" = {}; # gRPC
    };

    Env = [
      "SSL_CERT_FILE=/etc/ssl/certs/ca-bundle.crt"
    ];
  };
}
