#!/bin/sh
set -e
R=sioworker/sio-cli
BIN=${SIO_BIN:-$HOME/.local/bin}
r= g= y= n=
if [ -t 1 ] && [ -z "$NO_COLOR" ]; then r='\033[31m' g='\033[32m' y='\033[33m' n='\033[0m'; fi
ok() { printf "$g✓$n %s\n" "$*"; }
warn() { printf "$y!$n %s\n" "$*"; }
die() { printf "$r✗$n %s\n" "$*" >&2; exit 1; }

if command -v curl >/dev/null; then get() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null; then get() { wget -qO "$2" "$1"; }
else die "need curl or wget"; fi

case $(uname -s) in Linux) os=linux ;; Darwin) os=darwin ;; *) die "unsupported os $(uname -s), grab a binary from https://github.com/$R/releases" ;; esac
case $(uname -m) in x86_64 | amd64) arch=amd64 ;; aarch64 | arm64) arch=arm64 ;; *) die "unsupported arch $(uname -m)" ;; esac

t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
get "https://github.com/$R/releases.atom" "$t/rel" # newest first, no api rate limit, autobuilds are prereleases so no /latest
tag=$(sed -n 's|.*/releases/tag/\([^"]*\)".*|\1|p' "$t/rel" | head -n1)
[ -n "$tag" ] || die "no release found"
f=sio-$os-$arch
get "https://github.com/$R/releases/download/$tag/$f" "$t/$f"
get "https://github.com/$R/releases/download/$tag/sha256sums.txt" "$t/sums"
if command -v sha256sum >/dev/null; then s=$(sha256sum "$t/$f"); else s=$(shasum -a 256 "$t/$f"); fi
grep -q "^${s%% *}  $f\$" "$t/sums" || die "checksum mismatch for $f"
mkdir -p "$BIN"
cp "$t/$f" "$BIN/sio" && chmod 755 "$BIN/sio"
ok "installed sio $tag to $BIN/sio"

raw=https://raw.githubusercontent.com/$R/$tag/completions
comp() { # shell, dir, src, dst
	command -v "$1" >/dev/null || return 0
	mkdir -p "$2" && get "$raw/$3" "$2/$4" && ok "$1 completions"
}
comp fish "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions" sio.fish sio.fish
comp bash "${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions" sio.bash sio
comp zsh "${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions" _sio _sio

case :$PATH: in *:"$BIN":*) ;; *) warn "$BIN is not in PATH, add it in your shell rc" ;; esac
ok "next: sio add-host <name> <domain>"
