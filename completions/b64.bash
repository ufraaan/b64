_b64() {
	local cur prev opts
	COMPREPLY=()
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[COMP_CWORD-1]}"

	opts="encode decode completion -w --url --raw --ignore-garbage --check --base32 --base16 --help"

	case "${prev}" in
	encode|decode)
		_filedir
		return 0
		;;
	completion)
		COMPREPLY=($(compgen -W "bash zsh fish" -- "${cur}"))
		return 0
		;;
	-w)
		COMPREPLY=()
		return 0
		;;
	*)
		COMPREPLY=($(compgen -W "${opts}" -- "${cur}"))
		return 0
		;;
	esac
} && complete -F _b64 b64
