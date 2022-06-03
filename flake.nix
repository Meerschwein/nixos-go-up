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
        (_: prev: {nixos-go-up = prev.callPackage ./nix/nixos-go-up.nix {};})
        (_: _: {nixos-shell = inputs.nixos-shell.defaultPackage.${system};})
      ];
    };
  in rec
  {
    devShell.${system} = pkgs.mkShell {
      packages = with pkgs; [
        go_1_17

        # Go tools
        gopls
        gopkgs
        go-outline
        gotests
        delve
        go-tools
        gomodifytags
        impl
        golangci-lint

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
    defaultApp.${system} = apps.${system}.default;
    nixosConfigurations.vm = lib.makeOverridable lib.nixosSystem {
      inherit system pkgs lib;
      modules = [
        inputs.nixos-shell.nixosModules.nixos-shell
        ./nix/vm.nix
      ];
    };
  };
}
