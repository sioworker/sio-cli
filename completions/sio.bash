_sio() {
	local cur=${COMP_WORDS[COMP_CWORD]} w="${COMP_WORDS[*]:1:COMP_CWORD-1}" o
	case $w in
	"") o='config ping info tree upload probs subs' ;;
	config) o='hosts lang quotes' ;;
	"config quotes") o='on off' ;;
	"config hosts") o='add rm main token' ;;
	"config hosts add") o='--main' ;;
	"config hosts rm" | "config hosts main" | "config hosts token" | ping | info | tree) o=$(sio config hosts 2>/dev/null | cut -c3- | cut -d' ' -f1) ;;
	"config lang") o="$(sio config lang 2>/dev/null | cut -c3- | cut -d' ' -f1) auto" ;;
	upload\ * | up\ *) COMPREPLY=($(compgen -f -- "$cur")); return ;;
	*) return ;;
	esac
	COMPREPLY=($(compgen -W "$o" -- "$cur"))
}
complete -F _sio sio
