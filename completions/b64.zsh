#compdef b64

_b64() {
	local -a opts
	opts=(
		'(-w)'{-w,}'[wrap encoded output at COLUMNS (0 = no wrap)]:columns'
		'--url[use URL-safe alphabet]'
		'--raw[URL-safe alphabet without padding]'
		'--ignore-garbage[skip non-alphabet characters during decode]'
		'--check[validate input format without decoding]'
		'--base32[use base32 encoding]'
		'--base16[use base16 (hex) encoding]'
		'--help[show help]'
		'encode:encode input: _files'
		'decode:decode input: _files'
		'completion:generate completions:(bash zsh fish)'
	)
	_describe -t commands 'b64' opts
}

_b64 "$@"
