let
  sources = import ./nix/sources.nix;
  pkgs = import sources.nixpkgs {};
in
pkgs.mkShell {
  name = "scripts-shell";
  buildInputs = with pkgs; [
    semver-tool
    kubectl
    kubernetes-helm
    yq-go
    chart-testing
    go_1_19
    kubernetes-controller-tools
  ];
}
