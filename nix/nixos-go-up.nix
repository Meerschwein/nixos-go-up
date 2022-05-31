{
  buildGoModule,
  makeWrapper,
  lib,
  cryptsetup,
  openssl,
  parted,
  yubikey-personalization,
  callPackage,
}: let
  pbkdf2Sha512 = callPackage ./pbkdf2-sha512.nix {};
in
  buildGoModule rec {
    name = "nixos-go-up";
    src = ./..;
    vendorSha256 = "sha256-SLk0qiI2PDoakFva9b9aDURCGs7aimqYTbO66RxhP6I=";
    doCheck = false;

    nativeBuildInputs = [makeWrapper];

    wrapperPath = lib.makeBinPath [
      cryptsetup
      openssl
      parted
      yubikey-personalization

      pbkdf2Sha512
    ];

    postFixup = ''
      # Ensure all dependencies are in PATH
      wrapProgram $out/bin/${name} --prefix PATH : "${wrapperPath}"
    '';
  }