{
  pkgs,
  lib,
  modulesPath,
  ...
}: let
  GB = n: n * 1024;
  username = "meer";
in {
  nix = {
    nixPath = ["nixpkgs=${pkgs.path}"];
    package = pkgs.nixFlakes;
    extraOptions = "experimental-features = nix-command flakes";
  };

  virtualisation = {
    cores = 12;
    diskSize = GB 20;
    memorySize = GB 8;
    graphics = true;

    #useBootLoader = true;
    writableStoreUseTmpfs = false;
    emptyDiskImages = [
      (GB 20)
    ];

    bootDevice = "/dev/vda";
  };

  nixos-shell.mounts = {
    mountHome = false;
    mountNixProfile = false;
  };

  environment.systemPackages = with pkgs; [
    nixos-go-up
  ];

  users.mutableUsers = false;
  users.users.${username} = {
    isNormalUser = true;
    home = "/home/${username}";
    shell = pkgs.fish;
    useDefaultShell = false;
    extraGroups = ["wheel" "docker" "networkmanager" "adbusers"];
    hashedPassword = "$6$00000000$xhcjnloTbvLFvT.Gbn2bPqS5EDW2Wn7x6jEt.dEihnw/ocXZRu/R732RzeA1x52U50VecumRYc/HbIPWIHTZD.";
  };
}
