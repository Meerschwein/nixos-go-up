{nixpkgs ? import <nixpkgs> {}}: let
  inherit (nixpkgs) stdenv pkgs;
  nixos-go-up = nixpkgs.callPackage ./assets/nixos-go-up.nix {};
in
  stdenv.mkDerivation {
    name = "nixos-go-up";
    buildInputs = with pkgs; [
      nixos-go-up
    ];

    shellHook = ''
      sudo nixos-go-up
    '';
  }
