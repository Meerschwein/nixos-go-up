{
  buildGoModule,
  makeWrapper,
  lib,
  parted,
}:
buildGoModule rec {
  name = "nixos-go-up";
  src = ./..;
  vendorSha256 = "sha256-Vxe6e9ezLXfBXW1eWCI2PQs7YYw/ESHcJAyCP8QWwVo=";
  doCheck = false;

  nativeBuildInputs = [makeWrapper];

  wrapperPath = lib.makeBinPath [
    parted
  ];

  postFixup = ''
    # Ensure all dependencies are in PATH
    wrapProgram $out/bin/${name} --prefix PATH : "${wrapperPath}"
  '';
}
