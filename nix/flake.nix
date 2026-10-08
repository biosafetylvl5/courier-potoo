{
  description = "Python environment with Courier";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { nixpkgs, ... }:
    let
      system = "aarch64-darwin";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        packages = with pkgs; [
          python312
          uv
          git
        ];

        shellHook = ''
          if [ ! -f .venv/bin/courier ]; then
            uv venv .venv --python ${pkgs.python312}/bin/python
            uv pip install --python .venv/bin/python \
              "data-courier @ git+https://github.com/CIRA-GEOIPS/courier.git"
          fi

          source .venv/bin/activate
        '';
      };
    };
}
