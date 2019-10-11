{ pkgs ? import <nixpkgs> {} }:
with pkgs;

assert lib.versionAtLeast go.version "1.11";

buildGoPackage rec {
  name = "best-zone-for-spot-${version}";
  version = "0.0.1";
  goPackagePath = "github.com/PsyanticY/best-zone-for-spot";

  nativeBuildInputs = [ makeWrapper ];

  goDeps = ./deps.nix;
  src = ./.;

  postInstall = with stdenv; let
    binPath = lib.makeBinPath [ nix-prefetch-git go ];
  in ''
    wrapProgram $bin/bin/best-zone-for-spot --prefix PATH : ${binPath}
  '';

  meta = with stdenv.lib; {
    description = "Check best spot capacity in a given region";
    homepage = https://github.com/PsyanticY/best-zone-for-spot;
    license = licenses.mit;
  };

}