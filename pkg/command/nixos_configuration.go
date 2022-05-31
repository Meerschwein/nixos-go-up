package command

type Replacement struct {
	Bootloader           string
	GrubDevice           string
	Hostname             string
	Timezone             string
	NetworkingInterfaces string
	Desktopmanager       string
	KeyboardLayout       string
	Username             string
	PasswordHash         string
}

func NixOSConfiguration() string {
	return `
{ config, pkgs, ... }:

{
  # Enable Flake Support
  nix = {
    package = pkgs.nixFlakes;
    extraOptions = "experimental-features = nix-command flakes";
  };

  imports = [
    # Include the results of the hardware scan.
    ./hardware-configuration.nix
  ];

  # Use the GRUB 2 boot loader for BIOS and systemd-boot for UEFI
  {{ .Bootloader }}

  boot.loader.grub.device = "{{ .GrubDevice }}";

  networking.hostName = "{{ .Hostname }}";
  # networking.wireless.enable = true;  # Enables wireless support via wpa_supplicant.

  time.timeZone = "{{ .Timezone }}";

  # The global useDHCP flag is deprecated, therefore explicitly set to false here.
  # Per-interface useDHCP will be mandatory in the future, so this generated config
  # replicates the default behaviour.
  networking.useDHCP = false;
  {{ .NetworkingInterfaces }}

  # Enable the X11 windowing system.
  services.xserver.enable = true;

  # Enable the Desktop Environment.
  {{ .Desktopmanager }}

  # Configure keymap in X11
  services.xserver.layout = "{{ .KeyboardLayout }}";

  users.users."{{ .Username }}" = {
    isNormalUser = true;
    extraGroups = [ "wheel" "networkmanager" ];
    hashedPassword = "{{ .PasswordHash }}";
  };

  # List packages installed in system profile. To search, run:
  # $ nix search wget
  environment.systemPackages = with pkgs; [
    vim
    git
  ];

  # This value determines the NixOS release from which the default
  # settings for stateful data, like file locations and database versions
  # on your system were taken. It‘s perfectly fine and recommended to leave
  # this value at the release version of the first install of this system.
  # Before changing this value read the documentation for this option
  # (e.g. man configuration.nix or on https://nixos.org/nixos/options.html).
  system.stateVersion = "21.11"; # Did you read the comment?

}`
}
