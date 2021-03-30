{ pkgs ? import <nixpkgs> {} }:
{
  devEnv = pkgs.stdenv.mkDerivation {
    name = "dev";
    buildInputs = with pkgs; [
      stdenv
      go
    ];
    shellHook = ''
    go env | grep GOROOT

    if [ -f local.env ]; then
        set -o allexport
        source local.env
        set +o allexport
    fi
  '';
  };
}
