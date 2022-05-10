{ nixpkgs ? import <nixpkgs> { } }:

let
  inherit (nixpkgs) callPackage pkgs stdenv;
  pbkdf2Sha512 = callPackage ./assets/pbkdf2-sha512.nix { };
  nixos-go-up = callPackage ./assets/nixos-go-up.nix { };
in
stdenv.mkDerivation {
  name = "yubikey-luks-setup";
  buildInputs = with pkgs; [
    cryptsetup
    openssl
    parted
    yubikey-personalization
    
    pbkdf2Sha512
    nixos-go-up
  ];

  shellHook = ''
    sudo nixos-go-up
  '';

  inherit (pkgs) cryptsetup openssl yubikey-personalization;
}
