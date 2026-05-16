{
  description = "btrman - a modern TUI for Linux manual pages";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachSystem [ "x86_64-linux" "aarch64-linux" ] (system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;

        version = if self ? shortRev then self.shortRev else "dev";
        runtimePath = lib.makeBinPath [
          pkgs.xclip
          pkgs.xsel
          pkgs.wl-clipboard
        ];

        btrman = pkgs.buildGoModule {
          pname = "btrman";
          inherit version;
          src = self;

          subPackages = [ "cmd/btrman" ];
          vendorHash = "sha256-taYRFUBfRI1jcZPlpAp+DPr/QsP+U19mcSrdfVNIgu4=";

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram $out/bin/btrman \
              --prefix PATH : ${runtimePath}
          '';

          meta = {
            description = "Modern Bubble Tea terminal UI for browsing Linux manual pages";
            homepage = "https://github.com/ruloweb/btrman";
            license = lib.licenses.mit;
            mainProgram = "btrman";
            platforms = lib.platforms.linux;
          };
        };

        btrmanApp = {
          type = "app";
          program = lib.getExe btrman;
          meta = {
            description = "Run btrman, a modern TUI for Linux manual pages";
          };
        };
      in
      {
        packages = {
          default = btrman;
          btrman = btrman;
        };

        apps = {
          default = btrmanApp;
          btrman = btrmanApp;
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.pkg-config
            pkgs.xclip
            pkgs.xsel
            pkgs.wl-clipboard
          ];

          GOROOT = "${pkgs.go}/share/go";
        };
      });
}
