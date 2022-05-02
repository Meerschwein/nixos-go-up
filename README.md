```bash
echo "
{ config, pkgs, ... }:
{
  nix = {
    package = pkgs.nixFlakes;
    extraOptions = ''
      experimental-features = nix-command flakes
    '';
  };

  imports = [
    <nixpkgs/nixos/modules/installer/cd-dvd/installation-cd-graphical-gnome.nix>
  ];
}
" | sudo tee /etc/nixos/configuration.nix &&
sudo nixos-rebuild switch &&
sudo nix run https://github.com/Meerschwein/nixos-go-up/archive/refs/heads/main.zip
```