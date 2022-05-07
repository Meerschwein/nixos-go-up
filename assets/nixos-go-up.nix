{ buildGoModule }:

buildGoModule {
  name = "nixos-go-up";
  src = ./..;
  vendorSha256 = "sha256-4CQOqvQmbFltlmdQDV2dyLzXcRRvMrva6irtJCtfwFU=";
  doCheck = false;
}
