{
  description = "Caddy Gateway";

  inputs = {
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = {...} @ inputs:
    inputs.flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["aarch64-darwin" "aarch64-linux" "x86_64-darwin" "x86_64-linux"];

      imports = [
        inputs.treefmt-nix.flakeModule
      ];

      perSystem = {system, ...}: let
        pkgs = import inputs.nixpkgs {inherit system;};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_22
            gofumpt
            golangci-lint
            gotools

            kubectl
            kubernetes-controller-tools
            kustomize
            operator-sdk
          ];
        };

        treefmt = {
          projectRootFile = "flake.nix";

          programs = {
            # Enable alejandra, a Nix formatter.
            alejandra.enable = true;
            # Enable deadnix, a Nix linter/formatter that removes un-used Nix code.
            deadnix.enable = true;
            # Enable gofumpt, a Go formatter.
            gofumpt = {
              enable = true;
              extra = true;
            };
            # Enable prettier, a multipurpose formatter.
            prettier = {
              enable = true;
              includes = [
                "*.md"
              ];
            };
            # Enable shellcheck, a shell script linter.
            shellcheck.enable = true;
            # Enable shfmt, a shell script formatter.
            shfmt = {
              enable = true;
              indent_size = 0; # 0 causes shfmt to use tabs
            };
          };
        };
      };
    };
}
