{
  buildGoModule,
  makeWrapper,
  lib,
  cryptsetup,
  parted,
  yubikey-personalization,
  callPackage,
}:
buildGoModule rec {
  name = "nixos-go-up";
  src = ./..;
  vendorSha256 = "sha256-EbVgItCgIb3Qt22of+Um6bLc8SvqIhp9eXiA/rMoeNU=";
  doCheck = false;

  nativeBuildInputs = [makeWrapper];

  wrapperPath = lib.makeBinPath [
    cryptsetup
    parted
    yubikey-personalization
  ];

  postFixup = ''
    # Ensure all dependencies are in PATH
    wrapProgram $out/bin/${name} --prefix PATH : "${wrapperPath}"
  '';
}
