{
  description = "default";

  inputs = {
    nixpkgs-stable.url = "nixpkgs/nixos-21.11";
    nixpkgs-unstable.url = "nixpkgs/nixos-unstable";
  };

  outputs = { ... }@inputs:
    let
      system = "x86_64-linux";
      lib = inputs.nixpkgs-stable.lib;

      overlay-unstable = _: _: {
        unstable = import inputs.nixpkgs-unstable {
          inherit system;
        };
      };

      nixos-go-up =
        { buildGoModule }: buildGoModule {
          name = "nixos-go-up";
          src = ./.;
          vendorSha256 = "sha256-m0xRLabHKNEtZIcYWFFnIuswnZowXINzFGk5XKa/DaQ=";
        };

      pkgs = import inputs.nixpkgs-stable {
        inherit system;
        overlays = [
          overlay-unstable
          (_: super: { nixos-go-up = super.callPackage nixos-go-up { }; })
        ];
      };

    in
    rec
    {
      devShell.${system} = pkgs.mkShell {
        packages = with pkgs; [
          gopls
          gopkgs
          go-outline
          gotests
          delve
          go-tools
          gofumpt
          gomodifytags
          impl
          go_1_17
        ];
      };
      defaultApp.${system} = {
        type = "app";
        program = "${pkgs.nixos-go-up}/bin/nixos-go-up";
      };
    };

}
