{
  pkgs,
  lib,
  modulesPath,
  ...
}: let
  GB = n: n * 1024;
in {
  nix.package = pkgs.nixFlakes;
  nix.extraOptions = ''
    experimental-features = nix-command flakes
  '';
  imports = [
    #"${modulesPath}/installer/cd-dvd/installation-cd-minimal.nix"
  ];

  virtualisation = {
    cores = 12;
    diskSize = GB 20;
    memorySize = GB 8;
    #graphics = true;

    #useBootLoader = true;
  };

  nixos-shell.mounts = {
    mountHome = false;
    extraMounts = {
      "/root" = {
        target = ./.;
        cache = "none";
      };
    };
  };

  environment.systemPackages = with pkgs; [
    nixos-go-up
  ];
}
