{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url     = "github:NixOS/nixpkgs/release-24.05";
  };

  outputs = { self, flake-utils, nixpkgs }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = (import nixpkgs) { inherit system; };
      in {
        devShell = pkgs.mkShell {
          buildInputs = with pkgs; [
            google-cloud-sdk go go-tools gofumpt gopls gore hugo
            nodePackages.vscode-langservers-extracted
          ];

          GOROOT = "${pkgs.go}/share/go";
        };
      }
    );
}
