let
  sources = import ./nix/sources.nix;
  pkgs = import sources.nixpkgs {};
in
pkgs.mkShell {
  name = "scripts-shell";
  buildInputs = with pkgs; [
    chart-testing
    ginkgo
    git
    go_1_19
    golint
    kubectl
    kubernetes-controller-tools
    kubernetes-helm
    make
    minikube
    semver-tool
    yq-go
  ];
}
