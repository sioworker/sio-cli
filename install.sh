#!/bin/sh
set -e
R=sioworker/sio-cli
BIN=${SIO_BIN:-$HOME/.local/bin}
BIN=${BIN%/}
r= g= y= c= d= n=
if [ -t 1 ] && [ -z "$NO_COLOR" ]; then r='\033[31m' g='\033[32m' y='\033[33m' c='\033[36m' d='\033[2m' n='\033[0m'; fi
ok() { printf "$g✓$n %s\n" "$*"; }
warn() { printf "$y!$n %s\n" "$*"; }
die() { printf "\n$r✗$n %s\n" "$*" >&2; exit 1; }

if command -v curl >/dev/null; then get() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null; then get() { wget -qO "$2" "$1"; }
else die "need curl or wget"; fi

bar() { # url, out: get w/ progress bar when theres a tty + curl
	if [ ! -t 2 ] || ! command -v curl >/dev/null; then get "$1" "$2"; return; fi
	tot=$(curl -fsSLI "$1" 2>/dev/null | tr -d '\r' | awk 'tolower($1)=="content-length:"{v=$2} END{print v+0}')
	get "$1" "$2" &
	p=$!
	while kill -0 $p 2>/dev/null; do draw "$2"; sleep 0.1; done
	wait $p || die "download failed: $1"
	draw "$2"
	printf '\n' >&2
}

draw() { # _ vars, sh has no locals and f/s are taken
	_s=0
	[ -f "$1" ] && _s=$(($(wc -c <"$1")))
	if [ "$tot" -gt 0 ]; then
		_p=$((_s * 100 / tot)) _f=$((_s * 30 / tot)) _o= _i=0
		while [ $_i -lt 30 ]; do
			if [ $_i -lt $_f ]; then _o="$_o█"; else _o="$_o░"; fi
			_i=$((_i + 1))
		done
		printf "\r  $c%s$n %3d%% $d%s$n" "$_o" $_p "$(awk "BEGIN{printf \"%.1f/%.1f MB\", $_s/1048576, $tot/1048576}")" >&2
	else
		printf "\r  $d%s$n" "$(awk "BEGIN{printf \"%.1f MB\", $_s/1048576}")" >&2
	fi
}

case $(uname -s) in Linux) os=linux ;; Darwin) os=darwin ;; *) die "unsupported os $(uname -s), grab a binary from https://github.com/$R/releases" ;; esac
case $(uname -m) in x86_64 | amd64) arch=amd64 ;; aarch64 | arm64) arch=arm64 ;; *) die "unsupported arch $(uname -m)" ;; esac

t=$(mktemp -d) tty=
trap 'rm -rf "$t"; [ -n "$tty" ] && stty "$tty" </dev/tty; true' EXIT
get "https://github.com/$R/releases.atom" "$t/rel" # newest first, no api rate limit, autobuilds are prereleases so no /latest
tag=$(sed -n 's|.*/releases/tag/\([^"]*\)".*|\1|p' "$t/rel" | head -n1)
[ -n "$tag" ] || die "no release found"
f=sio-$os-$arch
printf "$c↓$n %s $d%s$n\n" "$f" "$tag"
bar "https://github.com/$R/releases/download/$tag/$f" "$t/$f"
get "https://github.com/$R/releases/download/$tag/sha256sums.txt" "$t/sums"
if command -v sha256sum >/dev/null; then s=$(sha256sum "$t/$f"); else s=$(shasum -a 256 "$t/$f"); fi
grep -q "^${s%% *}  $f\$" "$t/sums" || die "checksum mismatch for $f"
su=
if ! mkdir -p "$BIN" 2>/dev/null || [ ! -w "$BIN" ]; then # eg SIO_BIN=/usr/local/bin
	if command -v doas >/dev/null; then su=doas # i personally use doas
	elif command -v sudo >/dev/null; then su=sudo
	else die "cant write to $BIN and theres no doas or sudo"; fi
	warn "$BIN needs root, using $su"
fi
$su mkdir -p "$BIN"
$su cp "$t/$f" "$BIN/sio" && $su chmod 755 "$BIN/sio"
ok "installed sio $tag to $BIN/sio"

s1=fish s2=bash s3=zsh i=1 me=${SHELL##*/}
while [ $i -le 3 ]; do # has = installed, on = picked, def on for installed ones
	eval "nm=\$s$i"
	if command -v "$nm" >/dev/null; then eval "has$i=1 on$i=1"; else eval "has$i=0 on$i=0"; fi
	i=$((i + 1))
done
case $me in fish | zsh) on2=0 ;; esac # bash is on every box, dont want it unless its ur shell

lst() {
	i=1
	while [ $i -le 3 ]; do
		eval "nm=\$s$i on=\$on$i has=\$has$i"
		m='  ' xb='[ ]' no=
		[ $i = "$cur" ] && m="$c❯$n "
		[ "$on" = 1 ] && xb="$g[x]$n"
		[ "$has" = 0 ] && no=" $d(not installed)$n"
		[ "$nm" = "$me" ] && no=" $d(your shell)$n"
		printf "\r\033[K  $m$xb %s$no\n" "$nm"
		i=$((i + 1))
	done
}

if [ -n "${SIO_COMP+x}" ]; then # SIO_COMP="fish zsh" or none, skips the picker
	on1=0 on2=0 on3=0
	for w in $SIO_COMP; do case $w in fish) on1=1 ;; bash) on2=1 ;; zsh) on3=1 ;; esac; done
elif [ -t 1 ] && (: </dev/tty) 2>/dev/null; then # curl | sh eats stdin, so keys come from /dev/tty
	tty=$(stty -g </dev/tty)
	stty -icanon -echo min 1 time 0 </dev/tty
	cur=1 e=$(printf '\033')
	printf "$c?$n %s $d%s$n\n" "which completions?" "(↑↓ move, space toggle, enter ok)"
	lst
	while :; do
		k=$(dd bs=1 count=1 2>/dev/null </dev/tty)
		if [ "$k" = "$e" ]; then # arrow = esc [ A, cap the wait so a lone esc doesnt hang
			stty min 0 time 1 </dev/tty
			k=$(dd bs=2 count=1 2>/dev/null </dev/tty)
			stty min 1 time 0 </dev/tty
			[ -z "$k" ] && k=esc # lone esc, ignore it
		fi
		case $k in
		"[A" | k) [ $cur -gt 1 ] && cur=$((cur - 1)) ;;
		"[B" | j) [ $cur -lt 3 ] && cur=$((cur + 1)) ;;
		" ") eval "on$cur=\$((1 - on$cur))" ;;
		1 | 2 | 3) cur=$k && eval "on$cur=\$((1 - on$cur))" ;;
		"") break ;;
		esac
		printf '\033[3A'
		lst
	done
	stty "$tty" </dev/tty
	tty= cur=
	printf '\033[3A'
	lst
fi

raw=https://raw.githubusercontent.com/$R/$tag/completions
comp() { # shell, dir, src, dst
	mkdir -p "$2" && get "$raw/$3" "$2/$4" && ok "$1 completions"
}
[ "$on1" = 1 ] && comp fish "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions" sio.fish sio.fish
[ "$on2" = 1 ] && comp bash "${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions" sio.bash sio
[ "$on3" = 1 ] && comp zsh "${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions" _sio _sio
[ "$on1$on2$on3" = 000 ] && warn "no completions installed"

case :$PATH: in *:"$BIN":*) ;; *) warn "$BIN is not in PATH, add it in your shell rc" ;; esac
ok "next:"
ok "	- sio lang <lang>"
ok "	- sio add-host <name> <domain>"
