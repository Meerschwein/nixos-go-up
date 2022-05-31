{
  description = "default";

  inputs = {
    nixpkgs-stable.url = "nixpkgs/nixos-22.05";
    nixos-shell.url = "github:Mic92/nixos-shell";
  };

  outputs = {...} @ inputs: let
    system = "x86_64-linux";
    lib = inputs.nixpkgs-stable.lib;

    pkgs = import inputs.nixpkgs-stable {
      inherit system;
      overlays = [
        (_: prev: {nixos-go-up = prev.callPackage ./assets/nixos-go-up.nix {};})
        (_: _: {nixos-shell = inputs.nixos-shell.defaultPackage.${system};})
      ];
    };
  in rec
  {
    devShell.${system} = pkgs.mkShell {
      packages = with pkgs; [
        # Go tools
        gopls
        gopkgs
        go-outline
        gotests
        delve
        go-tools
        gomodifytags
        impl
        go_1_17

        # Formatting
        treefmt
        gofumpt
        alejandra

        nixos-shell
      ];
    };
    apps.${system}.default = {
      type = "app";
      program = "${pkgs.nixos-go-up}/bin/nixos-go-up";
    };
    nixosConfigurations.vm = lib.makeOverridable lib.nixosSystem {
      inherit system pkgs lib;
      modules = [
        inputs.nixos-shell.nixosModules.nixos-shell
        ./vm.nix
      ];
    };
  };
}
