{ buildGoModule }:

buildGoModule {
  name = "nixos-go-up";
  src = ./..;
  vendorSha256 = "sha256-j1rRvKzqxVux2LeqKzabjEhRd7QPB202fg85B790hbk=";
  doCheck = false;
}
