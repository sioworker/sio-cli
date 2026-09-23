{
	description = "CLI for submitting to OIOIOI (sio2) instances";

	inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

	outputs = { self, nixpkgs }:
		let
			each = f: nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ] (s: f nixpkgs.legacyPackages.${s});
		in {
			packages = each (pkgs: rec {
				sio = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) { # go.mod needs 1.27, def go is older
					pname = "sio-cli";
					version = "0-${self.shortRev or "dirty"}";
					src = self;
					vendorHash = "sha256-fxxp7ECuMMmLw7L5/lPi7lfmJM2p+Te9AqXm5Xed3G0=";
					subPackages = [ "cmd/sio" ];
					ldflags = [ "-s" ];
					nativeBuildInputs = [ pkgs.installShellFiles ];
					postInstall = ''
						installShellCompletion --cmd sio --bash completions/sio.bash --fish completions/sio.fish --zsh completions/_sio
					'';
					meta = {
						description = "CLI for submitting to OIOIOI (sio2) instances";
						homepage = "https://github.com/sioworker/sio-cli";
						license = pkgs.lib.licenses.asl20;
						mainProgram = "sio";
					};
				};
				default = sio;
			});
			devShells = each (pkgs: {
				default = pkgs.mkShell { packages = [ pkgs.go_1_27 pkgs.gopls ]; };
			});
		};
}
