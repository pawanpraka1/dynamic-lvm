let
  sources = import ./nix/sources.nix;
  pkgs = import sources.nixpkgs {};
in
pkgs.mkShell {
  name = "scripts-shell";
  buildInputs = with pkgs; [
    (minikube.override { withQemu = true; } )
    chart-testing
    git
    go_1_19
    kubectl
    kubernetes-controller-tools
    kubernetes-helm
    semver-tool
    yq-go
  ];
}
