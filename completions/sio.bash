_sio() {
	local cur=${COMP_WORDS[COMP_CWORD]} cmd=${COMP_WORDS[1]}
	if ((COMP_CWORD == 1)); then
		COMPREPLY=($(compgen -W 'add-host rm-host hosts main token ping upload probs subs' -- "$cur"))
		return
	fi
	case $cmd in
	add-host) ((COMP_CWORD == 2)) && COMPREPLY=($(compgen -W '--main' -- "$cur")) ;;
	rm-host | main | token | ping) ((COMP_CWORD == 2)) && COMPREPLY=($(compgen -W "$(sio hosts 2>/dev/null | cut -c3- | cut -d' ' -f1)" -- "$cur")) ;;
	upload | up) ((COMP_CWORD >= 3)) && COMPREPLY=($(compgen -f -- "$cur")) ;;
	esac
}
complete -F _sio sio
